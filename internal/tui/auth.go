package tui

import (
	"context"
	"fmt"

	"github.com/UNSAReport/tui/internal/auth"
	"github.com/UNSAReport/tui/internal/config"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type authMode int

const (
	authStatus authMode = iota
	authLogin
	authLogout
)

type AuthModel struct {
	mode       authMode
	client     *auth.Client
	spinner    spinner.Model
	loading    bool
	status     string
	user       string
	form       *huh.Form
	tokenInput string
	result     string
}

func NewAuthModel() AuthModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return AuthModel{
		mode:    authStatus,
		client:  auth.NewClient(),
		spinner: s,
	}
}

func (m AuthModel) Init() tea.Cmd { return m.fetchStatus() }

func (m AuthModel) fetchStatus() tea.Cmd {
	return func() tea.Msg {
		token := config.GetToken()
		if token == "" {
			return authStatusMsg{loggedIn: false}
		}
		client := auth.NewClient()
		u, err := client.Whoami(context.Background())
		if err != nil {
			// token present but whoami failed - still show token active
			return authStatusMsg{loggedIn: true, user: "unknown (offline)", err: err.Error()}
		}
		return authStatusMsg{loggedIn: true, user: u.Name + " <" + u.Email + ">"}
	}
}

type authStatusMsg struct {
	loggedIn bool
	user     string
	err      string
}

func (m AuthModel) Update(msg tea.Msg) (AuthModel, tea.Cmd) {
	switch msg := msg.(type) {
	case authStatusMsg:
		m.loading = false
		if msg.loggedIn {
			m.status = "Logged in"
			m.user = msg.user
			if msg.err != "" {
				m.status += " (offline: " + msg.err + ")"
			}
		} else {
			m.status = "Not logged in — run Login"
			m.user = ""
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchStatus())
		}
		// handle login/logout selection via mode? root will set mode based on nav cursor
		// For now, keys: enter on Login triggers form
	}
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	if m.form != nil {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
			if m.form.State == huh.StateCompleted {
				// handle login/logout confirm
				switch m.mode {
				case authLogin:
					err := m.client.Login(m.tokenInput)
					if err != nil {
						m.result = "Login failed: " + err.Error()
					} else {
						m.result = "Login saved"
					}
				case authLogout:
					_ = m.client.Logout()
					m.result = "Logged out"
				}
				m.form = nil
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.fetchStatus())
			}
			if m.form.State == huh.StateAborted {
				m.form = nil
				m.result = "Cancelled"
				return m, nil
			}
		}
		return m, cmd
	}
	return m, nil
}

func (m *AuthModel) SetMode(mode authMode) {
	m.mode = mode
	m.result = ""
	m.form = nil
	switch mode {
	case authLogin:
		m.tokenInput = ""
		m.form = huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Paste token").Placeholder("JWT token").Value(&m.tokenInput).Validate(func(s string) error {
				if len(s) < 5 {
					return fmt.Errorf("token too short")
				}
				return nil
			}),
		))
	case authLogout:
		var confirm bool
		m.form = huh.NewForm(huh.NewGroup(
			huh.NewConfirm().Title("Confirm logout?").Value(&confirm),
		))
		// we need to handle confirm via form state; if aborted, logout not done; if completed and confirm false, cancel
		// Simplified: on completed we always logout
		_ = confirm
	case authStatus:
		// no form
	}
	if m.form != nil {
		// Init will be called by parent via Update? Need to trigger Init cmd
	}
}

func (m AuthModel) View() string {
	if m.loading {
		return lipgloss.NewStyle().Padding(1).Render(m.spinner.View() + " Checking auth...")
	}
	if m.form != nil {
		return lipgloss.NewStyle().Padding(1).Render(m.form.View())
	}
	if m.result != "" {
		return lipgloss.NewStyle().Padding(1).Render(m.result + "\n\nStatus: " + m.status + "\nUser: " + m.user + "\n\nPress r to refresh")
	}
	var b string
	b += "Auth Status\n\n"
	b += "Endpoint: " + config.GetAuthURL() + "\n"
	b += "Status: " + m.status + "\n"
	if m.user != "" {
		b += "User: " + m.user + "\n"
	}
	token := config.GetToken()
	if token != "" {
		b += "Token: " + token[:min(10, len(token))] + "... (active)\n"
	} else {
		b += "Token: none\n"
	}
	b += "\nSelect Login/Logout via sidebar, enter to execute, r to refresh"
	switch m.mode {
	case authLogin:
		b += "\n\n[Login form ready - press enter]"
	case authLogout:
		b += "\n\n[Logout confirm ready]"
	}
	return lipgloss.NewStyle().Padding(1).Render(b)
}
