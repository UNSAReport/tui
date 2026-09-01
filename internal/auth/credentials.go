package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/UNSAReport/tui/internal/config"
	"github.com/zalando/go-keyring"
)

const (
	keyringService = config.KeyringService
	keyringUser    = config.KeyringUser
	patPrefix      = config.PATPrefix
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
	if v := os.Getenv(config.EnvXDGConfigHome); v != "" {
		return filepath.Join(v, config.AppDirName, config.CredentialsFileName)
	}
	if d, err := os.UserConfigDir(); err == nil && d != "" {
		return filepath.Join(d, config.AppDirName, config.CredentialsFileName)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".config", config.AppDirName, config.CredentialsFileName)
	}
	return ""
}


func writeAtomicCredentials(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, config.PermDirPrivate); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(path)
		if err2 := os.Rename(tmp, path); err2 != nil {
			return err2
		}
	}
	return nil
}
func NewFileStore() *FileStore {
	p := defaultCredentialsPath()
	if v := os.Getenv(config.EnvCredentialsPath); v != "" {
		p = v
	}
	return &FileStore{Path: p}
}

func (s *FileStore) Get() (*Credentials, error) {
	b, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in")
		}
		return nil, err
	}
	var c Credentials
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("corrupt credentials file %s: %w", s.Path, err)
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
	if s.Path == "" {
		return fmt.Errorf("credentials path unavailable")
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := writeAtomicCredentials(s.Path, b, config.PermFilePrivate); err != nil {
		return err
	}
	return nil
}

func (s *FileStore) Clear() error {
	if err := os.Remove(s.Path); err != nil && !os.IsNotExist(err) {
		return err
	}
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
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return nil, fmt.Errorf("corrupt keyring credentials: %w", err)
	}
	if c.PAT == "" {
		return nil, fmt.Errorf("not logged in")
	}
	return &c, nil
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

func NewStore() Store {
	return &KeyringStore{}
}

func GetTokenResolved(store Store) string {
	if v := os.Getenv(config.EnvToken); v != "" {
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
