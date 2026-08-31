package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "unsareport"
	keyringUser    = "pat"
	patPrefix      = "unsareport_pat_"
)

type Credentials struct {
	PAT       string            `json:"pat"`
	UserID    string            `json:"user_id,omitempty"`
	Email     string            `json:"email,omitempty"`
	Name      string            `json:"name,omitempty"`
	Roles     map[string]string `json:"roles,omitempty"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

type Store interface {
	Get() (*Credentials, error)
	Set(*Credentials) error
	Clear() error
}

type FileStore struct {
	Path string
}

func defaultCredentialsPath() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "unsareport", "credentials.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "unsareport", "credentials.json")
	}
	return filepath.Join(home, ".config", "unsareport", "credentials.json")
}

func legacyTokenPath() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "unsareport", "token")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "unsareport", "token")
	}
	return filepath.Join(home, ".config", "unsareport", "token")
}

func NewFileStore() *FileStore {
	p := defaultCredentialsPath()
	if v := os.Getenv("UNSAREP_CREDENTIALS_PATH"); v != "" {
		p = v
	}
	return &FileStore{Path: p}
}

func (s *FileStore) Get() (*Credentials, error) {
	b, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			if tok, err2 := os.ReadFile(legacyTokenPath()); err2 == nil && len(tok) > 0 {
				return &Credentials{PAT: strings.TrimSpace(string(tok)), CreatedAt: time.Now()}, nil
			}
			return nil, fmt.Errorf("not logged in")
		}
		return nil, err
	}
	var c Credentials
	if err := json.Unmarshal(b, &c); err != nil {
		s2 := strings.TrimSpace(string(b))
		if s2 != "" {
			return &Credentials{PAT: s2, CreatedAt: time.Now()}, nil
		}
		return nil, err
	}
	if c.PAT == "" {
		return nil, fmt.Errorf("not logged in")
	}
	return &c, nil
}

func (s *FileStore) Set(c *Credentials) error {
	if c == nil || c.PAT == "" {
		return fmt.Errorf("empty credentials")
	}
	if !strings.HasPrefix(c.PAT, patPrefix) {
		fmt.Fprintf(os.Stderr, "warning: PAT should start with %s\n", patPrefix)
	}
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}

func (s *FileStore) Clear() error {
	if err := os.Remove(s.Path); err != nil && !os.IsNotExist(err) {
		return err
	}
	_ = os.Remove(legacyTokenPath())
	return nil
}

type KeyringStore struct{}

func (k *KeyringStore) Get() (*Credentials, error) {
	raw, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("not logged in")
	}
	var c Credentials
	if err := json.Unmarshal([]byte(raw), &c); err == nil && c.PAT != "" {
		return &c, nil
	}
	return &Credentials{PAT: raw, CreatedAt: time.Now()}, nil
}

func (k *KeyringStore) Set(c *Credentials) error {
	if c == nil || c.PAT == "" {
		return fmt.Errorf("empty credentials")
	}
	if !strings.HasPrefix(c.PAT, patPrefix) {
		fmt.Fprintf(os.Stderr, "warning: PAT should start with %s\n", patPrefix)
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return keyring.Set(keyringService, keyringUser, string(b))
}

func (k *KeyringStore) Clear() error {
	err := keyring.Delete(keyringService, keyringUser)
	if err != nil && err != keyring.ErrNotFound {
		if strings.Contains(err.Error(), "unsupported") || strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return nil
}

type FallbackStore struct {
	Keyring *KeyringStore
	File    *FileStore
}

func NewStore() Store {
	return &FallbackStore{Keyring: &KeyringStore{}, File: NewFileStore()}
}

func (s *FallbackStore) Get() (*Credentials, error) {
	if c, err := s.Keyring.Get(); err == nil {
		return c, nil
	}
	return s.File.Get()
}

func (s *FallbackStore) Set(c *Credentials) error {
	if err := s.Keyring.Set(c); err == nil {
		_ = s.File.Set(c)
		return nil
	}
	return s.File.Set(c)
}

func (s *FallbackStore) Clear() error {
	_ = s.Keyring.Clear()
	return s.File.Clear()
}

func GetTokenResolved(store Store) string {
	if v := os.Getenv("UNSAREP_TOKEN"); v != "" {
		return strings.TrimSpace(v)
	}
	if v := os.Getenv("UNSAREP_PAT"); v != "" {
		return strings.TrimSpace(v)
	}
	if store == nil {
		store = NewStore()
	}
	c, err := store.Get()
	if err != nil || c == nil {
		return ""
	}
	return c.PAT
}
