package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/UNSAReport/tui/internal/config"
)

// TemplateInfo describes a template.
type TemplateInfo struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Path        string            `json:"path"`
	DistTags    map[string]string `json:"distTags,omitempty"`
	Versions    map[string]any    `json:"versions,omitempty"` // raw for fallback
	Tags        []string          `json:"tags,omitempty"`
}

// Client fetches registry data from Hono service with legacy fallback.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	CachePath  string
}

func NewClient() *Client {
	base := config.GetRegistryURL()
	// ensure no trailing slash
	base = strings.TrimSuffix(base, "/")
	cache := filepath.Join(xdgCacheDir(), "registry.json")
	return &Client{
		BaseURL:    base,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		CachePath:  cache,
	}
}

func xdgCacheDir() string {
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return filepath.Join(v, "unsareport")
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, ".cache", "unsareport")
	}
	return ".cache/unsareport"
}

// ListTemplates returns all templates. Tries Hono /v1/packages then fallback to cached registry.json.
func (c *Client) ListTemplates(ctx context.Context) ([]TemplateInfo, error) {
	// Try new API
	if c.BaseURL != "" && !strings.Contains(c.BaseURL, "github") {
		req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/v1/packages?limit=100", nil)
		if err == nil {
			req.Header.Set("User-Agent", "unsarep-tui")
			resp, err := c.HTTPClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					// Try generic decode: try both shapes
					var raw map[string]any
					if err := json.NewDecoder(resp.Body).Decode(&raw); err == nil {
						// attempt to extract packages array via second decode
						b, _ := json.Marshal(raw)
						var p2 struct {
							Packages []TemplateInfo `json:"packages"`
							Total    int            `json:"total"`
						}
						if err2 := json.Unmarshal(b, &p2); err2 == nil && len(p2.Packages) > 0 {
							// cache
							_ = os.MkdirAll(filepath.Dir(c.CachePath), 0o700)
							_ = os.WriteFile(c.CachePath, b, 0o600)
							return p2.Packages, nil
						}
						// Fallback shape: packages contain name/description
						if pkgs, ok := raw["packages"].([]any); ok {
							var out []TemplateInfo
							for _, pa := range pkgs {
								if m, ok := pa.(map[string]any); ok {
									ti := TemplateInfo{}
									if v, ok := m["name"].(string); ok {
										ti.Name = v
									}
									if v, ok := m["description"].(string); ok {
										ti.Description = v
									}
									if v, ok := m["displayName"].(string); ok && ti.Description == "" {
										ti.Description = v
									}
									out = append(out, ti)
								}
							}
							if len(out) > 0 {
								return out, nil
							}
						}
					}
				}
			}
		}
	}
	// Fallback to legacy registry.json (local cache or bundled)
	return c.listFromLegacy()
}

type legacyRegistryFile struct {
	Templates map[string]struct {
		Description string            `json:"description"`
		DistTags    map[string]string `json:"dist-tags"`
		Versions    map[string]struct {
			Path string `json:"path"`
		} `json:"versions"`
	} `json:"templates"`
}

func (c *Client) listFromLegacy() ([]TemplateInfo, error) {
	paths := []string{c.CachePath, "templates/registry.json", "../templates/registry.json", "../../templates/registry.json", "../../../templates/registry.json"}
	// Also walk up from cwd looking for templates/registry.json
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for range 6 {
			paths = append(paths, filepath.Join(dir, "templates/registry.json"))
			paths = append(paths, filepath.Join(dir, "templates", "templates", "registry.json"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	var lastErr error
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			lastErr = err
			continue
		}
		var lf legacyRegistryFile
		if err := json.Unmarshal(b, &lf); err != nil {
			lastErr = err
			continue
		}
		var out []TemplateInfo
		for name, entry := range lf.Templates {
			latest := ""
			if v, ok := entry.DistTags["latest"]; ok {
				latest = v
			}
			ti := TemplateInfo{
				Name:        name,
				Description: entry.Description,
				Version:     latest,
				DistTags:    entry.DistTags,
			}
			vers := make(map[string]any)
			for ver, v := range entry.Versions {
				vers[ver] = v.Path
			}
			ti.Versions = vers
			out = append(out, ti)
		}
		return out, nil
	}
	return nil, fmt.Errorf("registry unavailable: %v", lastErr)
}

// GetTemplate fetches single template metadata.
func (c *Client) GetTemplate(ctx context.Context, name string) (TemplateInfo, error) {
	name = strings.ToLower(name)
	if c.BaseURL != "" && !strings.Contains(c.BaseURL, "github") {
		u := fmt.Sprintf("%s/v1/packages/%s", c.BaseURL, url.PathEscape(name))
		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err == nil {
			req.Header.Set("User-Agent", "unsarep-tui")
			resp, err := c.HTTPClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				switch resp.StatusCode {
				case 200:
					var pkg TemplateInfo
					// Decode directly into TemplateInfo plus extra
					var raw map[string]any
					if err := json.NewDecoder(resp.Body).Decode(&raw); err == nil {
						b, _ := json.Marshal(raw)
						_ = json.Unmarshal(b, &pkg)
						if pkg.Name == "" {
							if v, ok := raw["name"].(string); ok {
								pkg.Name = v
							}
						}
						if pkg.Description == "" {
							if v, ok := raw["description"].(string); ok {
								pkg.Description = v
							}
						}
						// versions array
						if vs, ok := raw["versions"].([]any); ok {
							m := make(map[string]any)
							for _, v := range vs {
								if s, ok := v.(string); ok {
									m[s] = s
								}
							}
							pkg.Versions = m
						}
						if pkg.Name != "" {
							return pkg, nil
						}
					}
				case 404:
					return TemplateInfo{}, fmt.Errorf("template %q not found", name)
				}
			}
		}
	}
	// fallback legacy
	all, err := c.listFromLegacy()
	if err != nil {
		return TemplateInfo{}, err
	}
	for _, t := range all {
		if strings.EqualFold(t.Name, name) {
			return t, nil
		}
	}
	return TemplateInfo{}, fmt.Errorf("template %q not found", name)
}

// GetTemplateVersion resolves rangeSpec against available versions.
func (c *Client) GetTemplateVersion(ctx context.Context, name, rangeSpec string) (TemplateInfo, error) {
	info, err := c.GetTemplate(ctx, name)
	if err != nil {
		return TemplateInfo{}, err
	}
	// Need to fetch versions list for semver resolution if Versions missing dist-tags
	// Try to fetch /v1/packages/:name/versions for accurate list
	var versionsMap map[string]*semver.Version
	var distTags map[string]*semver.Version

	// Build from info.Versions + DistTags
	versionsMap = make(map[string]*semver.Version)
	for ver := range info.Versions {
		if v, err := semver.NewVersion(ver); err == nil {
			versionsMap[ver] = v
		}
	}
	distTags = make(map[string]*semver.Version)
	for tag, verStr := range info.DistTags {
		if v, err := semver.NewVersion(verStr); err == nil {
			distTags[tag] = v
		}
	}
	// If versionsMap empty, try fetching versions endpoint
	if len(versionsMap) == 0 && c.BaseURL != "" {
		u := fmt.Sprintf("%s/v1/packages/%s/versions", c.BaseURL, url.PathEscape(strings.ToLower(name)))
		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err == nil {
			req.Header.Set("User-Agent", "unsarep-tui")
			resp, err := c.HTTPClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					var payload struct {
						Versions []struct {
							Version string `json:"version"`
						} `json:"versions"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil {
						for _, v := range payload.Versions {
							if sv, err := semver.NewVersion(v.Version); err == nil {
								versionsMap[v.Version] = sv
							}
						}
					}
				}
			}
		}
	}
	// If still empty, fallback to single version entry
	if len(versionsMap) == 0 && info.Version != "" {
		if v, err := semver.NewVersion(info.Version); err == nil {
			versionsMap[info.Version] = v
			if _, ok := distTags["latest"]; !ok {
				distTags["latest"] = v
			}
		}
	}

	resolved, err := resolveVersionFromMap(versionsMap, distTags, rangeSpec)
	if err != nil {
		return TemplateInfo{}, err
	}
	info.Version = resolved.Original()
	return info, nil
}

// resolveVersionFromMap mirrors UNSAReport version.go logic.
func resolveVersionFromMap(available map[string]*semver.Version, distTags map[string]*semver.Version, rangeSpec string) (*semver.Version, error) {
	switch rangeSpec {
	case "latest", "":
		if latest, ok := distTags["latest"]; ok {
			return latest, nil
		}
		return nil, fmt.Errorf("no 'latest' dist-tag found")
	case "*":
		return resolveLatest(available)
	}
	constraint, err := semver.NewConstraint(rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("invalid version range %q: %w", rangeSpec, err)
	}
	var candidates []*semver.Version
	for _, v := range available {
		if ok, _ := constraint.Validate(v); ok {
			candidates = append(candidates, v)
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no version matching %q", rangeSpec)
	}
	// sort by string compare (semver string) to match legacy cmp.Compare
	// Use semver Compare
	// Find max
	max := candidates[0]
	for _, c := range candidates[1:] {
		if c.GreaterThan(max) {
			max = c
		}
	}
	return max, nil
}

func resolveLatest(versions map[string]*semver.Version) (*semver.Version, error) {
	var all []*semver.Version
	for _, v := range versions {
		if v.Prerelease() != "" {
			continue
		}
		all = append(all, v)
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("no stable versions found")
	}
	max := all[0]
	for _, v := range all[1:] {
		if v.GreaterThan(max) {
			max = v
		}
	}
	return max, nil
}
