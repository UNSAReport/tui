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
	path := filepath.Join(destDir, ConfigFileName)
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
	} else {
		return UnsareportConfig{}, false, nil
	}

	validMode := cfg.Mode == "" || cfg.Mode == "single" || cfg.Mode == "multi"
	if !validMode {
		return UnsareportConfig{}, found, fmt.Errorf("invalid mode %q in unsareport.json (must be \"single\" or \"multi\")", cfg.Mode)
	}
	if cfg.Mode == "multi" && len(cfg.Sessions) == 0 {
		return UnsareportConfig{}, found, fmt.Errorf("multi-mode requires at least one session in unsareport.json")
	}
	return cfg.WithDefaults(), found, nil
}

func (c UnsareportConfig) WithDefaults() UnsareportConfig {
	if c.Prepare.Input.SrcDir == "" {
		c.Prepare.Input.SrcDir = DefaultSrcDir
	}
	if c.Prepare.Output.SubmissionDir == "" {
		c.Prepare.Output.SubmissionDir = DefaultSubmissionDir
	}
	if c.Prepare.Input.ReportFile == "" {
		c.Prepare.Input.ReportFile = DefaultReportFile
	}
	if c.Capture.Prompt == "" {
		c.Capture.Prompt = DefaultPrompt
	}
	if c.Capture.Columns == 0 {
		c.Capture.Columns = DefaultColumns
	}
	if c.Capture.Colors == nil {
		c.Capture.Colors = map[string]string{
			"prompt":  ColorPrompt,
			"command": ColorCommand,
			"args":    ColorArgs,
			"reset":   ColorReset,
		}
	}
	if c.Capture.Columns <= 0 {
		c.Capture.Columns = DefaultColumns
	}
	if c.Capture.Rows <= 0 {
		c.Capture.Rows = DefaultRows
	}
	return c
}
func WriteConfig(destDir string, cfg UnsareportConfig) error {
	parts := strings.SplitN(Version, ".", 3)
	majorMinor := Version
	if len(parts) >= 2 {
		majorMinor = parts[0] + "." + parts[1]
	}
	cfg.Schema = fmt.Sprintf(SchemaBaseURL+"/v%s/schemas/unsareport.schema.json", majorMinor)
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	b = append(b, '\n')
	path := filepath.Join(destDir, ConfigFileName)
	if err := os.WriteFile(path, b, PermFilePublic); err != nil {
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
