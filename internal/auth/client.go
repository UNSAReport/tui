package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/UNSAReport/tui/internal/config"
)

type UserInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient() *Client {
	return &Client{
		BaseURL:    config.GetAuthURL(),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Whoami(ctx context.Context) (UserInfo, error) {
	token := config.GetToken()
	if token == "" {
		return UserInfo{}, fmt.Errorf("not logged in")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/auth/me", nil)
	if err != nil {
		return UserInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "unsarep-tui")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return UserInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return UserInfo{}, fmt.Errorf("whoami failed: %d", resp.StatusCode)
	}
	var u UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return UserInfo{}, err
	}
	return u, nil
}

func (c *Client) Login(token string) error {
	if token == "" {
		return fmt.Errorf("token must not be empty")
	}
	// Simple validation: token looks like JWT? At least non-empty
	if len(token) < 10 {
		return fmt.Errorf("token too short")
	}
	return config.SaveToken(token)
}

func (c *Client) Logout() error {
	return config.ClearToken()
}

func (c *Client) GetToken() string { return config.GetToken() }
func (c *Client) IsLoggedIn() bool { return c.GetToken() != "" }
