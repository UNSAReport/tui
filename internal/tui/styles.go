package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)
type Theme struct {
	Primary lipgloss.AdaptiveColor
	Muted   lipgloss.AdaptiveColor
	Success lipgloss.AdaptiveColor
	Warning lipgloss.AdaptiveColor
	Error   lipgloss.AdaptiveColor
	Border  lipgloss.AdaptiveColor
}

func defaultTheme() Theme {
	// Detect background to allow manual tweak if needed; AdaptiveColor handles auto.
	_ = lipgloss.HasDarkBackground()
	_ = termenv.HasDarkBackground()
	return Theme{
		Primary: lipgloss.AdaptiveColor{Light: "#1a2b4a", Dark: "#89b4fa"},
		Muted:   lipgloss.AdaptiveColor{Light: "#6c7086", Dark: "#9399b2"},
		Success: lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"},
		Warning: lipgloss.AdaptiveColor{Light: "#df8e1d", Dark: "#f9e2af"},
		Error:   lipgloss.AdaptiveColor{Light: "#d20f39", Dark: "#f38ba8"},
		Border:  lipgloss.AdaptiveColor{Light: "#ccd0da", Dark: "#45475a"},
	}
}

type Styles struct {
	Theme Theme

	TopBar       lipgloss.Style
	TopBarActive lipgloss.Style
	TopBarMuted  lipgloss.Style
	Sidebar      lipgloss.Style
	SidebarSel   lipgloss.Style
	Main         lipgloss.Style
	HelpBar      lipgloss.Style
	Border       lipgloss.Style
}

func newStyles(t Theme) Styles {
	return Styles{
		Theme: t,
		TopBar: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary).
			Padding(0, 1),
		TopBarActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(t.Primary).
			Padding(0, 1),
		TopBarMuted: lipgloss.NewStyle().
			Foreground(t.Muted).
			Padding(0, 1),
		Sidebar: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(0, 1),
		SidebarSel: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary),
		Main: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(0, 1),
		HelpBar: lipgloss.NewStyle().
			Foreground(t.Muted).
			Padding(0, 1),
		Border: lipgloss.NewStyle().
			BorderForeground(t.Border),
	}
}
