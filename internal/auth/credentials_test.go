package auth

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/UNSAReport/tui/internal/config"
)

func TestFileStorePermAndAtomic(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("UNSAREP_TOKEN", "")
	t.Setenv("UNSAREP_CREDENTIALS_PATH", "")
	_ = (&KeyringStore{}).Clear()
	fs := NewFileStore()
	cred := &Credentials{PAT: "unsareport_pat_test123", Email: "a@example.com", Name: "Alice", CreatedAt: time.Now()}
	if err := fs.Set(cred); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(fs.Path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != config.PermFilePrivate {
		t.Fatalf("perm %o want 0600", info.Mode().Perm())
	}
	dirInfo, _ := os.Stat(filepath.Dir(fs.Path))
	if runtime.GOOS != "windows" && dirInfo.Mode().Perm() != config.PermDirPrivate {
		t.Fatalf("dir perm %o want 0700", dirInfo.Mode().Perm())
	}
	got, err := fs.Get()
	if err != nil {
		t.Fatal(err)
	}
	if got.PAT != cred.PAT || got.Email != "a@example.com" {
		t.Fatalf("got %+v", got)
	}
	if err := fs.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Get(); err == nil {
		t.Fatal("should error after clear")
	}
}

func TestGetTokenResolvedEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("UNSAREP_TOKEN", "env_pat_123")
	_ = (&KeyringStore{}).Clear()
	fs := NewFileStore()
	_ = fs.Set(&Credentials{PAT: "unsareport_pat_file", CreatedAt: time.Now()})
	got := GetTokenResolved(fs)
	if got != "env_pat_123" {
		t.Fatalf("got %q want env", got)
	}
	t.Setenv("UNSAREP_TOKEN", "")
	_ = (&KeyringStore{}).Clear()
	got = GetTokenResolved(fs)
	if got != "unsareport_pat_file" {
		t.Fatalf("got %q want file", got)
	}
	_ = fs.Clear()
}

func TestFileStoreClear(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("UNSAREP_TOKEN", "")
	_ = (&KeyringStore{}).Clear()
	fs := NewFileStore()
	cred := &Credentials{PAT: "unsareport_pat_fb", CreatedAt: time.Now()}
	if err := fs.Set(cred); err != nil {
		t.Fatal(err)
	}
	got, err := fs.Get()
	if err != nil {
		t.Fatal(err)
	}
	if got.PAT != cred.PAT {
		t.Fatalf("got %q", got.PAT)
	}
	if err := fs.Clear(); err != nil {
		t.Fatal(err)
	}
}
