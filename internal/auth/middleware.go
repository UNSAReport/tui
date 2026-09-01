package auth

import (
	"context"
	"fmt"
	"os"

	"github.com/UNSAReport/tui/internal/config"
	"github.com/charmbracelet/huh"
)
func RequireAuth(ctx context.Context, c *Client, prompt bool) (*Credentials, error) {
	tok := ""
	if v := os.Getenv(config.EnvToken); v != "" {
		tok = v
	} else if c != nil {
		tok = c.GetToken()
	}
	if tok != "" {
		if cred, _, err := c.Status(ctx); err == nil {
			return cred, nil
		}
		if c.Store != nil {
			if cred, err := c.Store.Get(); err == nil {
				return cred, nil
			}
		}
		return &Credentials{PAT: tok}, nil
	}
	if !prompt {
		return nil, fmt.Errorf("not authenticated — run 'unsarep login'")
	}
	var doLogin bool
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title("Not authenticated. Login now?").Value(&doLogin),
	))
	if err := form.Run(); err != nil {
		return nil, err
	}
	if !doLogin {
		return nil, fmt.Errorf("authentication required")
	}
	cred, err := c.Login(ctx, false)
	if err != nil {
		return nil, err
	}
	return cred, nil
}
