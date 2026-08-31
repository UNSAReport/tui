package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type updateItem struct{ name, status string }

func (i updateItem) Title() string       { return i.name }
func (i updateItem) Description() string { return i.status }
func (i updateItem) FilterValue() string { return i.name }

type DocsUpdateModel struct {
	list          list.Model
	viewport      viewport.Model
	loading       bool
	width, height int
}

func NewDocsUpdateModel() DocsUpdateModel {
	l := list.New([]list.Item{
		updateItem{"report.typ", "M - modified"},
		updateItem{"lib.typ", "A - added"},
	}, list.NewDefaultDelegate(), 30, 10)
	l.Title = "Update — Check"
	vp := viewport.New(40, 10)
	vp.SetContent("Diff preview will appear here.\nPress y to apply, n to skip, a for all, f force")
	return DocsUpdateModel{list: l, viewport: vp}
}

func (m DocsUpdateModel) Init() tea.Cmd { return nil }

func (m DocsUpdateModel) Update(msg tea.Msg) (DocsUpdateModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(m.width/2, m.height-6)
		m.viewport.Width = m.width / 2
		m.viewport.Height = m.height - 6
	case tea.KeyMsg:
		switch msg.String() {
		case "y":
			return m, nil
		case "n":
			return m, nil
		case "a":
			return m, nil
		case "f":
			return m, nil
		case "r":
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m DocsUpdateModel) View() string {
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("#888")).Render("y apply, n skip, a all, q quit, f force, r retry")
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(m.width/2).Render(m.list.View()),
		lipgloss.JoinVertical(lipgloss.Left, m.viewport.View(), help),
	)
}

func (m *DocsUpdateModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.list.SetSize(w/2, h-6)
	m.viewport.Width = w / 2
	m.viewport.Height = h - 6
}

func (m DocsUpdateModel) Placeholder() string { return "Update placeholder" }
