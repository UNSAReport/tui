package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CaptureConfig struct {
	Columns        int               `json:"columns"`
	Rows           int               `json:"rows,omitempty"`
	FreezeFlags    []string          `json:"freezeFlags"`
	Prompt         string            `json:"prompt"`
	Colors         map[string]string `json:"colors"`
	CommandTimeout int               `json:"commandTimeout,omitempty"`
}

type PrepareInputConfig struct {
	SrcDir     string `json:"srcDir"`
	ReportFile string `json:"reportFile"`
}

type PrepareOutputConfig struct {
	SubmissionDir string `json:"submissionDir"`
	FileTemplate  string `json:"fileTemplate"`
	ReportWord    string `json:"reportWord"`
	CodeWord      string `json:"codeWord"`
}

type PrepareConfig struct {
	Input  PrepareInputConfig  `json:"input"`
	Output PrepareOutputConfig `json:"output"`
}

type UnsareportConfig struct {
	Schema          string                     `json:"$schema,omitempty"`
	Template        string                     `json:"template"`
	TemplateVersion string                     `json:"templateVersion,omitempty"`
	Mode            string                     `json:"mode"`
	LocalSource     string                     `json:"localSource,omitempty"`
	Sessions        []string                   `json:"sessions"`
	Capture         CaptureConfig              `json:"capture"`
	Prepare         PrepareConfig              `json:"prepare"`
	Components      map[string]string          `json:"components,omitempty"`
}

type ProjectContext struct {
	Root               string
	Config             UnsareportConfig
	IsProject          bool
	PreselectedSession string
}

var Version = "1.0.0"

func FindProjectRoot(startDir string) (string, UnsareportConfig, bool, error) {
	currentDir := startDir
	for {
		cfg, ok, err := ReadConfig(currentDir)
		if ok || err != nil {
			return currentDir, cfg, ok, err
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}
	return startDir, UnsareportConfig{}, false, nil
}

func ReadConfig(destDir string) (UnsareportConfig, bool, error) {
	path := filepath.Join(destDir, "unsareport.json")
	var cfg UnsareportConfig

	found := true
	b, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return UnsareportConfig{}, false, fmt.Errorf("read file: %w", err)
		}
		found = false
	}

	if found {
		if err := json.Unmarshal(b, &cfg); err != nil {
			return UnsareportConfig{}, true, fmt.Errorf("failed to parse unsareport.json: %w", err)
		}
	}

	if cfg.Prepare.Input.SrcDir == "" {
		cfg.Prepare.Input.SrcDir = "src"
	}
	if cfg.Prepare.Output.SubmissionDir == "" {
		cfg.Prepare.Output.SubmissionDir = "submission"
	}
	if cfg.Prepare.Input.ReportFile == "" {
		cfg.Prepare.Input.ReportFile = "report.typ"
	}
	if cfg.Capture.Prompt == "" {
		cfg.Capture.Prompt = "❯ "
	}
	if cfg.Capture.Columns == 0 {
		cfg.Capture.Columns = 120
	}
	if cfg.Capture.Colors == nil {
		cfg.Capture.Colors = map[string]string{
			"prompt":  "32",
			"command": "36",
			"args":    "33",
			"reset":   "0",
		}
	}
	if cfg.Capture.Columns <= 0 {
		cfg.Capture.Columns = 120
	}
	if cfg.Capture.Rows <= 0 {
		cfg.Capture.Rows = 500
	}
	validMode := cfg.Mode == "" || cfg.Mode == "single" || cfg.Mode == "multi"
	if !validMode {
		return UnsareportConfig{}, found, fmt.Errorf("invalid mode %q in unsareport.json (must be \"single\" or \"multi\")", cfg.Mode)
	}
	if cfg.Mode == "multi" && len(cfg.Sessions) == 0 {
		return UnsareportConfig{}, found, fmt.Errorf("multi-mode requires at least one session in unsareport.json")
	}
	return cfg, found, nil
}

func WriteConfig(destDir string, cfg UnsareportConfig) error {
	parts := strings.SplitN(Version, ".", 3)
	majorMinor := Version
	if len(parts) >= 2 {
		majorMinor = parts[0] + "." + parts[1]
	}
	cfg.Schema = fmt.Sprintf("https://raw.githubusercontent.com/UNSAReport/UNSAReport/v%s/schemas/unsareport.schema.json", majorMinor)
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	b = append(b, '\n')
	path := filepath.Join(destDir, "unsareport.json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

func DetectProject(startDir string) (*ProjectContext, error) {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		abs = startDir
	}
	root, cfg, ok, err := FindProjectRoot(abs)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &ProjectContext{IsProject: false, Root: abs}, nil
	}
	preselected := ""
	if abs != root {
		rel, err := filepath.Rel(root, abs)
		if err == nil {
			parts := strings.Split(rel, string(filepath.Separator))
			if len(parts) >= 1 {
				candidate := parts[0]
				for _, s := range cfg.Sessions {
					if s == candidate {
						preselected = candidate
						break
					}
				}
			}
		}
	}
	return &ProjectContext{
		Root:               root,
		Config:             cfg,
		IsProject:          true,
		PreselectedSession: preselected,
	}, nil
}
