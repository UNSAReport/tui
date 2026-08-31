package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type XDGConfig struct {
	APIURL      string `json:"apiUrl,omitempty"`
	RegistryURL string `json:"registryUrl,omitempty"`
	TokenPath   string `json:"tokenPath,omitempty"`
	Locale      string `json:"locale,omitempty"`
}

func xdgDir() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "unsareport")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".config/unsareport"
	}
	return filepath.Join(home, ".config", "unsareport")
}

func xdgConfigPath() string { return filepath.Join(xdgDir(), "config.json") }
func defaultTokenPath() string { return filepath.Join(xdgDir(), "token") }

func LoadXDGConfig() (*XDGConfig, error) {
	path := xdgConfigPath()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &XDGConfig{}, nil
		}
		return nil, fmt.Errorf("read xdg config: %w", err)
	}
	var cfg XDGConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse xdg config: %w", err)
	}
	return &cfg, nil
}

func SaveXDGConfig(cfg *XDGConfig) error {
	dir := xdgDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir xdg: %w", err)
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	path := xdgConfigPath()
	if err := os.WriteFile(path, append(b, '\n'), 0o600); err != nil {
		return fmt.Errorf("write xdg config: %w", err)
	}
	return nil
}

func GetRegistryURL() string {
	if v := os.Getenv("UNSAREP_REGISTRY_URL"); v != "" {
		return v
	}
	if v := os.Getenv("UNSAREP_REGISTRY"); v != "" {
		return v
	}
	cfg, _ := LoadXDGConfig()
	if cfg != nil && cfg.RegistryURL != "" {
		return cfg.RegistryURL
	}
	if v := os.Getenv("REGISTRY_URL"); v != "" {
		return v
	}
	return "https://registry.unsareport.org"
}

func GetAuthURL() string {
	if v := os.Getenv("UNSAREP_IDP_ISSUER"); v != "" {
		return v
	}
	if v := os.Getenv("IDP_ISSUER"); v != "" {
		return v
	}
	if v := os.Getenv("UNSAREP_API_URL"); v != "" {
		return v
	}
	if v := os.Getenv("UNSAREP_AUTH_URL"); v != "" {
		return v
	}
	cfg, _ := LoadXDGConfig()
	if cfg != nil && cfg.APIURL != "" {
		return cfg.APIURL
	}
	return "https://auth.unsareport.org"
}

func GetToken() string {
	if v := os.Getenv("UNSAREP_TOKEN"); v != "" {
		return v
	}
	cfg, _ := LoadXDGConfig()
	tokenPath := defaultTokenPath()
	if cfg != nil && cfg.TokenPath != "" {
		tokenPath = cfg.TokenPath
	}
	if v := os.Getenv("UNSAREP_TOKEN_PATH"); v != "" {
		tokenPath = v
	}
	b, err := os.ReadFile(tokenPath)
	if err != nil {
		return ""
	}
	return string(b)
}

func SaveToken(token string) error {
	cfg, _ := LoadXDGConfig()
	path := defaultTokenPath()
	if cfg != nil && cfg.TokenPath != "" {
		path = cfg.TokenPath
	}
	if v := os.Getenv("UNSAREP_TOKEN_PATH"); v != "" {
		path = v
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		return fmt.Errorf("write token: %w", err)
	}
	return nil
}

func ClearToken() error {
	cfg, _ := LoadXDGConfig()
	path := defaultTokenPath()
	if cfg != nil && cfg.TokenPath != "" {
		path = cfg.TokenPath
	}
	if v := os.Getenv("UNSAREP_TOKEN_PATH"); v != "" {
		path = v
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func GetLocale() string {
	if v := os.Getenv("UNSAREP_LOCALE"); v != "" {
		return v
	}
	if v := os.Getenv("LANG"); v != "" {
		_ = v
	}
	cfg, _ := LoadXDGConfig()
	if cfg != nil && cfg.Locale != "" {
		return cfg.Locale
	}
	return ""
}

func GetDest() string      { return os.Getenv("UNSAREP_DEST") }
func GetSession() string   { return os.Getenv("UNSAREP_SESSION") }
func GetLocal() string     { return os.Getenv("UNSAREP_LOCAL") }
func GetFreezeFlags() string { return os.Getenv("UNSAREP_FREEZE_FLAGS") }
