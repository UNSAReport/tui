package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestXDGStore(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("UNSAREP_TOKEN", "")
	t.Setenv("UNSAREP_REGISTRY_URL", "")

	cfg := &XDGConfig{RegistryURL: "https://example.com", APIURL: "https://idp.example.com", Locale: "es"}
	if err := SaveXDGConfig(cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadXDGConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.RegistryURL != cfg.RegistryURL {
		t.Fatalf("registry %q", loaded.RegistryURL)
	}
	if GetRegistryURL() != cfg.RegistryURL {
		t.Fatalf("GetRegistryURL %q", GetRegistryURL())
	}
	t.Setenv("UNSAREP_REGISTRY_URL", "https://override.com")
	if GetRegistryURL() != "https://override.com" {
		t.Fatal("env override failed")
	}
	t.Setenv("UNSAREP_REGISTRY_URL", "")

	if err := SaveToken("secret123"); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(filepath.Join(tmp, AppDirName, TokenFileName))
	if runtime.GOOS != "windows" && info.Mode().Perm() != PermFilePrivate {
		t.Fatalf("perm %o", info.Mode().Perm())
	}
	if tok := GetToken(); tok != "secret123" {
		t.Fatalf("token %q", tok)
	}
	t.Setenv("UNSAREP_TOKEN", "envtok")
	if GetToken() != "envtok" {
		t.Fatal("env token override")
	}
	t.Setenv("UNSAREP_TOKEN", "")
	if err := ClearToken(); err != nil {
		t.Fatal(err)
	}
	if GetToken() != "" {
		t.Fatal("token should be empty after clear")
	}
}
