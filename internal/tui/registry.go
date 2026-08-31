package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/UNSAReport/tui/internal/i18n"
	"github.com/UNSAReport/tui/internal/registry"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type registryItem struct {
	title, desc string
	info        registry.TemplateInfo
}

func (i registryItem) Title() string       { return i.title }
func (i registryItem) Description() string { return i.desc }
func (i registryItem) FilterValue() string { return i.title + " " + i.desc }

type registryFetchedMsg struct {
	templates []registry.TemplateInfo
	err       error
}

type registryMode int

const (
	registryBrowse registryMode = iota
	registrySearch
	registryDetails
)

type RegistryModel struct {
	mode          registryMode
	client        *registry.Client
	spinner       spinner.Model
	list          list.Model
	textInput     textinput.Model
	templates     []registry.TemplateInfo
	filtered      []registry.TemplateInfo
	selected      *registry.TemplateInfo
	loading       bool
	err           string
	width, height int
	focusSearch   bool
}

func NewRegistryModel() RegistryModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "Search templates..."
	ti.CharLimit = 50
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 40, 15)
	l.Title = i18n.T("category.templates") + " — " + i18n.T("function.browse")
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	return RegistryModel{
		client:    registry.NewClient(),
		spinner:   s,
		list:      l,
		textInput: ti,
		loading:   true,
		mode:      registryBrowse,
	}
}

func (m RegistryModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchCmd())
}

func (m RegistryModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		templates, err := m.client.ListTemplates(ctx)
		return registryFetchedMsg{templates: templates, err: err}
	}
}

func (m RegistryModel) Update(msg tea.Msg) (RegistryModel, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h := m.height - 6
		if h < 10 {
			h = 10
		}
		w := m.width/2 - 4
		if w < 30 {
			w = 30
		}
		m.list.SetSize(w, h)
		_ = h
	case registryFetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.templates = msg.templates
		m.filtered = msg.templates
		items := make([]list.Item, len(msg.templates))
		for i, t := range msg.templates {
			desc := t.Description
			if desc == "" {
				desc = t.Version
			}
			items[i] = registryItem{title: t.Name, desc: desc, info: t}
		}
		m.list.SetItems(items)
		return m, nil
	case tea.KeyMsg:
		if m.mode == registrySearch {
			switch msg.String() {
			case "esc":
				m.mode = registryBrowse
				m.focusSearch = false
				m.textInput.Blur()
				return m, nil
			case "enter":
				m.mode = registryBrowse
				m.focusSearch = false
				m.textInput.Blur()
				return m, nil
			}
		}
		switch msg.String() {
		case "r":
			if m.err != "" {
				m.loading = true
				m.err = ""
				return m, tea.Batch(m.spinner.Tick, m.fetchCmd())
			}
		case "/":
			if m.mode == registryBrowse {
				m.mode = registrySearch
				m.focusSearch = true
				m.textInput.Focus()
				return m, textinput.Blink
			}
		case "enter":
			if m.mode == registryBrowse && !m.loading && m.err == "" {
				if it, ok := m.list.SelectedItem().(registryItem); ok {
					m.selected = &it.info
					m.mode = registryDetails
					return m, nil
				}
			}
		case "esc", "b":
			if m.mode == registryDetails {
				m.mode = registryBrowse
				m.selected = nil
				return m, nil
			}
		}
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	if m.mode == registrySearch {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
		q := strings.ToLower(m.textInput.Value())
		if q == "" {
			m.filtered = m.templates
		} else {
			var out []registry.TemplateInfo
			for _, t := range m.templates {
				if strings.Contains(strings.ToLower(t.Name), q) || strings.Contains(strings.ToLower(t.Description), q) {
					out = append(out, t)
				}
			}
			m.filtered = out
		}
		items := make([]list.Item, len(m.filtered))
		for i, t := range m.filtered {
			items[i] = registryItem{title: t.Name, desc: t.Description, info: t}
		}
		m.list.SetItems(items)
		return m, tea.Batch(cmds...)
	}
	if m.mode == registryBrowse {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	return m, tea.Batch(cmds...)
}

func (m RegistryModel) View() string {
	if m.loading {
		return lipgloss.NewStyle().Padding(1).Render(m.spinner.View() + " Loading templates...")
	}
	if m.err != "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Padding(1).Render(fmt.Sprintf("Error: %s\nPress [r] to retry", m.err))
	}
	switch m.mode {
	case registryDetails:
		if m.selected != nil {
			var b strings.Builder
			b.WriteString(lipgloss.NewStyle().Bold(true).Render(m.selected.Name) + "\n\n")
			b.WriteString("Description: " + m.selected.Description + "\n")
			if m.selected.Version != "" {
				b.WriteString("Latest: " + m.selected.Version + "\n")
			}
			if len(m.selected.DistTags) > 0 {
				b.WriteString("Dist-tags:\n")
				for k, v := range m.selected.DistTags {
					fmt.Fprintf(&b, "  %s: %s\n", k, v)
				}
			}
			if len(m.selected.Versions) > 0 {
				b.WriteString("Versions: ")
				var vers []string
				for k := range m.selected.Versions {
					vers = append(vers, k)
				}
				b.WriteString(strings.Join(vers, ", "))
				b.WriteString("\n")
			}
			b.WriteString("\nPress [esc/b] to back, [enter] to install (Phase 2)")
			return lipgloss.NewStyle().Padding(1).Render(b.String())
		}
	case registrySearch:
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Padding(1, 1, 0, 1).Render("Search: "+m.textInput.View()+" (esc to cancel, enter to apply)"),
			m.list.View(),
		)
	}
	helpLine := lipgloss.NewStyle().Foreground(lipgloss.Color("#888")).Render("Press / to search, enter for details, r to retry")
	return lipgloss.JoinVertical(lipgloss.Left, m.list.View(), helpLine)
}

func (m RegistryModel) IsLoading() bool { return m.loading }
func (m *RegistryModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	h2 := h - 6
	if h2 < 10 {
		h2 = 10
	}
	w2 := w - 4
	if w2 < 30 {
		w2 = 30
	}
	m.list.SetSize(w2, h2)
}
