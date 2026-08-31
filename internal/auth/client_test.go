package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateAndStore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer unsareport_pat_good" {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path == "/api/auth/me" || r.URL.Path == "/v1/auth/me" {
			json.NewEncoder(w).Encode(map[string]any{
				"user": map[string]any{"id": "u1", "name": "Alice", "email": "alice@example.com"},
				"roles": map[string]string{"admin": "true"},
			})
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("UNSAREP_TOKEN", "")
	fs := NewFileStore()
	if fs.Path != filepath.Join(tmp, "unsareport", "credentials.json") {
		fs.Path = filepath.Join(tmp, "unsareport", "credentials.json")
	}
	client := NewClientWithConfig(ClientConfig{IDPIssuer: srv.URL, HTTPClient: srv.Client(), Store: &FallbackStore{Keyring: &KeyringStore{}, File: fs}})
	cred, err := client.ValidateAndStore(context.Background(), "unsareport_pat_good")
	if err != nil {
		t.Fatal(err)
	}
	if cred.Email != "alice@example.com" || cred.UserID != "u1" {
		t.Fatalf("cred %+v", cred)
	}
	if _, err := os.Stat(fs.Path); err != nil {
		t.Fatal(err)
	}
	stored, user, err := client.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "alice@example.com" {
		t.Fatalf("user %+v", user)
	}
	if stored.PAT != "unsareport_pat_good" {
		t.Fatal("stored mismatch")
	}
}

func TestValidateAndStoreInvalid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("UNSAREP_TOKEN", "")
	fs := &FileStore{Path: filepath.Join(tmp, "unsareport", "credentials.json")}
	client := NewClientWithConfig(ClientConfig{IDPIssuer: srv.URL, HTTPClient: srv.Client(), Store: fs})
	_, err := client.ValidateAndStore(context.Background(), "badpat")
	if err == nil {
		t.Fatal("should fail")
	}
}

func TestLoginWithTokenFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/me" {
			w.WriteHeader(404)
			return
		}
		json.NewEncoder(w).Encode(UserInfo{ID: "u2", Name: "Bob", Email: "bob@example.com"})
	}))
	defer srv.Close()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("UNSAREP_TOKEN", "")
	fs := &FileStore{Path: filepath.Join(tmp, "unsareport", "credentials.json")}
	client := NewClientWithConfig(ClientConfig{IDPIssuer: srv.URL, HTTPClient: &http.Client{Timeout: 2 * time.Second}, Store: fs})
	cred, err := client.LoginWithToken(context.Background(), "unsareport_pat_fallback")
	if err != nil {
		t.Fatal(err)
	}
	if cred.UserID != "u2" {
		t.Fatalf("cred %+v", cred)
	}
	_ = time.Now()
}
