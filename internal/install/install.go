package install

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/UNSAReport/tui/internal/config"
	"github.com/UNSAReport/tui/internal/manifest"
	"github.com/UNSAReport/tui/internal/registry"
)

type Options struct {
	TemplateArg string // name[@range]
	Dest        string
	Session     string
	Local       string // local dir alternative
}

func parseTemplateArg(arg string) (name, rangeSpec string) {
	if idx := strings.LastIndex(arg, "@"); idx != -1 {
		return arg[:idx], arg[idx+1:]
	}
	return arg, ""
}

// Execute installs template into dest.
func Execute(ctx context.Context, opt Options) error {
	name, rangeSpec := parseTemplateArg(opt.TemplateArg)
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("template name must not be empty")
	}
	if rangeSpec != "" {
		// validate semver constraint early to give inline error
		// use registry resolve logic via client; if invalid, it will error
	}
	dest := opt.Dest
	if dest == "" {
		var err error
		dest, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("mkdir dest: %w", err)
	}
	// Resolve version via registry client
	client := registry.NewClient()
	info, err := client.GetTemplateVersion(ctx, name, rangeSpec)
	if err != nil {
		return err
	}
	// Fetch files
	files, err := fetchFiles(ctx, client, info, opt.Local)
	if err != nil {
		return err
	}
	// Load manifest
	manifestData, ok := files["manifest.json"]
	if !ok {
		return fmt.Errorf("manifest.json not found in template")
	}
	man, err := manifest.LoadAndValidateManifest(manifestData)
	if err != nil {
		return err
	}
	// Determine entries to install
	var entries []manifest.Entry
	isMulti := man.Mode == "multi"
	if isMulti {
		me, err := man.GetMultiEntries()
		if err != nil {
			return err
		}
		entries = append(entries, me.Root...)
		// For create, if session provided, substitute lab entry? For now handle both:
		if opt.Session != "" {
			// lab files with destination substitution: if dest contains {lab} placeholder? legacy substituteLab replaces Dest prefix?
			// Simplified: prefix lab files with session dir
			for _, e := range me.LabFiles {
				ne := e
				// If dest is like "l1/report.typ" or just file, we prefix session?
				// Legacy: substituteLab replaces Dest template variable; we approximate by joining session
				ne.Dest = filepath.ToSlash(filepath.Join(opt.Session, e.Dest))
				entries = append(entries, ne)
			}
		} else {
			// No session - skip lab files for multi without session (root only)
			_ = me.LabFiles
		}
		entries = manifest.ExpandDirEntries(files, entries)
	} else {
		single, err := man.GetSingleEntries()
		if err != nil {
			return err
		}
		entries = manifest.ExpandDirEntries(files, single)
	}

	// Write files
	for _, e := range entries {
		if e.Kind == manifest.KindDir {
			// dir entries themselves not written, only expanded files
			continue
		}
		data, ok := files[e.Src]
		if !ok {
			return fmt.Errorf("file %q not found in template", e.Src)
		}
		destPath := filepath.Join(dest, filepath.FromSlash(e.Dest))
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return err
		}
	}

	// Write unsareport.json
	cfg := config.UnsareportConfig{
		Template:        name,
		TemplateVersion: info.Version,
		Mode:            man.Mode,
	}
	if isMulti {
		// If session provided, sessions = [session]; else empty? For multi, need at least one session.
		if opt.Session != "" {
			cfg.Sessions = []string{opt.Session}
		} else {
			cfg.Sessions = []string{}
		}
		if opt.Local != "" {
			cfg.LocalSource = opt.Local
		}
	}
	// Apply defaults via WriteConfig? But WriteConfig stamps schema; we use config.WriteConfig which applies defaults? Actually ReadConfig applies defaults, Write just writes.
	if err := config.WriteConfig(dest, cfg); err != nil {
		return err
	}
	// Write lockfile (simplified placeholder)
	lock := map[string]any{
		"template":        name,
		"version":         info.Version,
		"entries":         entries,
		"components":      man.GetComponents(),
	}
	b, _ := json.MarshalIndent(lock, "", "  ")
	_ = os.WriteFile(filepath.Join(dest, ".unsareport.lock"), append(b, '\n'), 0o644)

	return nil
}

func fetchFiles(ctx context.Context, client *registry.Client, info registry.TemplateInfo, localDir string) (map[string][]byte, error) {
	if localDir != "" {
		return loadLocal(localDir)
	}
	candidates := []string{
		filepath.Join("templates", "templates", info.Name, info.Version),
		filepath.Join("..", "templates", "templates", info.Name, info.Version),
		filepath.Join("../../templates", "templates", info.Name, info.Version),
		filepath.Join("../../../templates", "templates", info.Name, info.Version),
		filepath.Join("templates", info.Name, info.Version),
		filepath.Join("..", "templates", info.Name, info.Version),
	}
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for range 6 {
			candidates = append(candidates, filepath.Join(dir, "templates", "templates", info.Name, info.Version))
			candidates = append(candidates, filepath.Join(dir, "templates", info.Name, info.Version))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return loadLocal(p)
		}
	}
	return nil, fmt.Errorf("template files not found locally for %s@%s (tried %v) and remote download not implemented; use --local", info.Name, info.Version, candidates)
}

func loadLocal(dir string) (map[string][]byte, error) {
	out := make(map[string][]byte)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no files in local dir %q", dir)
	}
	return out, nil
}
