package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/UNSAReport/tui/internal/config"
)

type UserInfo struct {
	ID    string            `json:"id"`
	Name  string            `json:"name"`
	Email string            `json:"email"`
	Roles map[string]string `json:"roles,omitempty"`
}

type ClientConfig struct {
	IDPIssuer  string
	HTTPClient *http.Client
	Store      Store
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Store      Store
}

func NewClient() *Client {
	return NewClientWithConfig(ClientConfig{
		IDPIssuer:  config.GetAuthURL(),
		HTTPClient: &http.Client{Timeout: config.AuthTimeout},
		Store:      NewStore(),
	})
}

func NewClientWithConfig(cfg ClientConfig) *Client {
	iss := cfg.IDPIssuer
	if iss == "" {
		iss = config.GetAuthURL()
	}
	iss = strings.TrimSuffix(iss, "/")
	httpc := cfg.HTTPClient
	if httpc == nil {
		httpc = &http.Client{Timeout: config.AuthTimeout}
	}
	store := cfg.Store
	if store == nil {
		store = NewStore()
	}
	return &Client{BaseURL: iss, HTTPClient: httpc, Store: store}
}

func websiteBase() string {
	if v := os.Getenv(config.EnvWebsiteURL); v != "" {
		return strings.TrimSuffix(v, "/")
	}
	return ""
}

func (c *Client) resolveToken() string {
	if v := strings.TrimSpace(os.Getenv(config.EnvToken)); v != "" {
		return v
	}
	if c.Store != nil {
		if cred, err := c.Store.Get(); err == nil && cred != nil && cred.PAT != "" {
			return cred.PAT
		}
	}
	return strings.TrimSpace(config.GetToken())
}

func (c *Client) Whoami(ctx context.Context) (UserInfo, error) {
	u, _, err := c.whoamiWithToken(ctx, c.resolveToken())
	return u, err
}

func (c *Client) whoamiWithToken(ctx context.Context, token string) (UserInfo, map[string]string, error) {
	if strings.TrimSpace(token) == "" {
		return UserInfo{}, nil, fmt.Errorf("not logged in")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/v1/me", nil)
	if err != nil {
		return UserInfo{}, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "unsarep-tui")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return UserInfo{}, nil, err
	}
	body, _ := readAll(resp)
	if resp.StatusCode != 200 {
		return UserInfo{}, nil, fmt.Errorf("whoami failed: %d body %s", resp.StatusCode, string(body))
	}
	var u UserInfo
	if err := json.Unmarshal(body, &u); err == nil && u.ID != "" {
		return u, u.Roles, nil
	}
	var env struct {
		User  *UserInfo         `json:"user"`
		Roles map[string]string `json:"roles"`
		Data  *UserInfo         `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err == nil {
		if env.User != nil && env.User.ID != "" {
			if env.Roles != nil {
				env.User.Roles = env.Roles
			}
			return *env.User, env.Roles, nil
		}
		if env.Data != nil && env.Data.ID != "" {
			return *env.Data, env.Data.Roles, nil
		}
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err == nil {
		if um, ok := m["user"].(map[string]any); ok {
			b2, _ := json.Marshal(um)
			var u2 UserInfo
			if err := json.Unmarshal(b2, &u2); err == nil && u2.ID != "" {
				if r, ok := m["roles"].(map[string]any); ok {
					roles := map[string]string{}
					for k, v := range r {
						if s, ok := v.(string); ok {
							roles[k] = s
						}
					}
					u2.Roles = roles
				}
				return u2, u2.Roles, nil
			}
		}
	}
	return UserInfo{}, nil, fmt.Errorf("unexpected whoami shape: %s", string(body))
}

func readAll(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return buf, nil
}

func (c *Client) ValidateAndStore(ctx context.Context, pat string) (*Credentials, error) {
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return nil, fmt.Errorf("empty token")
	}
	u, roles, err := c.whoamiWithToken(ctx, pat)
	if err != nil {
		return nil, err
	}
	cred := &Credentials{
		PAT:       pat,
		UserID:    u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Roles:     roles,
		CreatedAt: time.Now(),
	}
	if u.Roles != nil && cred.Roles == nil {
		cred.Roles = u.Roles
	}
	if c.Store != nil {
		if err := c.Store.Set(cred); err != nil {
			return nil, err
		}
	} else if err := config.SaveToken(pat); err != nil {
		return nil, err
	}
	return cred, nil
}

func (c *Client) Status(ctx context.Context) (*Credentials, *UserInfo, error) {
	tok := c.resolveToken()
	if tok == "" {
		return nil, nil, fmt.Errorf("not logged in")
	}
	var stored *Credentials
	if c.Store != nil {
		if cc, err := c.Store.Get(); err == nil {
			stored = cc
		}
	}
	u, roles, err := c.whoamiWithToken(ctx, tok)
	if err != nil {
		if v := strings.TrimSpace(os.Getenv(config.EnvToken)); v != "" && stored != nil && v == tok && stored.PAT != v {
			return nil, nil, err
		}
		if stored != nil {
			return stored, nil, fmt.Errorf("offline: %w", err)
		}
		return nil, nil, err
	}
	u.Roles = roles
	if stored == nil {
		stored = &Credentials{PAT: tok, UserID: u.ID, Email: u.Email, Name: u.Name, Roles: roles, CreatedAt: time.Now()}
	} else {
		stored.UserID = u.ID
		stored.Email = u.Email
		stored.Name = u.Name
		stored.Roles = roles
	}
	return stored, &u, nil
}

func (c *Client) Login(ctx context.Context, noBrowser bool) (*Credentials, error) {
	state, err := GenerateState()
	if err != nil {
		return nil, err
	}
	cb := NewCallbackServer(state)
	cbURL, err := cb.Start()
	if err != nil {
		return nil, err
	}
	defer cb.Close()

	website := websiteBase()
	if website == "" {
		return nil, fmt.Errorf("website URL not configured: set %s", config.EnvWebsiteURL)
	}
	authURL := fmt.Sprintf("%s/auth/login?tui_callback=%s&state=%s", website, url.QueryEscape(cbURL), url.QueryEscape(state))

	if noBrowser || IsHeadless() {
		fmt.Printf("Open this URL in your browser:\n  %s\n\nWaiting for callback at %s (timeout 5m)...\n", authURL, cbURL)
		res, err := cb.Wait(config.CallbackTimeout)
		if err == nil && res.PAT != "" {
			return c.ValidateAndStore(ctx, res.PAT)
		}
		return nil, fmt.Errorf("no callback received; paste PAT via 'unsarep login --token <PAT>'")
	}

	if err := OpenBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\nOpen this URL:\n  %s\n\n", err, authURL)
	}
	fmt.Printf("Opened browser to %s\nWaiting for login (timeout 5m)...\n", authURL)

	res, err := cb.Wait(config.CallbackTimeout)
	if err != nil {
		return nil, err
	}
	if res.PAT == "" {
		return nil, fmt.Errorf("callback did not contain pat")
	}
	return c.ValidateAndStore(ctx, res.PAT)
}

func (c *Client) LoginWithToken(ctx context.Context, token string) (*Credentials, error) {
	return c.ValidateAndStore(ctx, token)
}

func (c *Client) GetToken() string { return c.resolveToken() }
func (c *Client) IsLoggedIn() bool { return c.GetToken() != "" }

func (c *Client) Logout() error {
	tok := c.resolveToken()
	if tok != "" {
		_ = c.revokePat(tok)
	}
	if c.Store != nil {
		if err := c.Store.Clear(); err != nil {
			return err
		}
	}
	return config.ClearToken()
}

func (c *Client) revokePat(tok string) error {
	req, err := http.NewRequest("POST", c.BaseURL+"/v1/logout", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("User-Agent", "unsarep-tui")
	resp, err := c.HTTPClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
	return nil
}
