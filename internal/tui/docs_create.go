package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/UNSAReport/tui/internal/install"
	"github.com/UNSAReport/tui/internal/registry"
)

type docsCreateFetchedMsg struct {
	templates []registry.TemplateInfo
	err       error
}

type DocsCreateModel struct {
	templates []registry.TemplateInfo
	loading   bool
	err       string
	spinner   spinner.Model
	form      *huh.Form
	templateArg string
	version     string
	dest        string
	local       string
	useLocal    bool
	session     string
	showForm    bool
	result      string
	project     *ProjectContext
}

func NewDocsCreateModel(project *ProjectContext) DocsCreateModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	cwd, _ := os.Getwd()
	dest := cwd
	if project != nil && project.IsProject {
		dest = project.Root
	}
	return DocsCreateModel{
		loading: true,
		spinner: s,
		dest:    dest,
		project: project,
	}
}

func (m DocsCreateModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchTemplates())
}

func (m DocsCreateModel) fetchTemplates() tea.Cmd {
	return func() tea.Msg {
		client := registry.NewClient()
		templates, err := client.ListTemplates(context.Background())
		return docsCreateFetchedMsg{templates: templates, err: err}
	}
}

func (m DocsCreateModel) Update(msg tea.Msg) (DocsCreateModel, tea.Cmd) {
	switch msg := msg.(type) {
	case docsCreateFetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.templates = msg.templates
		m.buildForm()
		m.showForm = true
		if m.form != nil {
			return m, m.form.Init()
		}
		return m, nil
	case tea.KeyMsg:
		if m.err != "" && msg.String() == "r" {
			m.loading = true
			m.err = ""
			return m, tea.Batch(m.spinner.Tick, m.fetchTemplates())
		}
	}
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	if m.showForm && m.form != nil {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
			if m.form.State == huh.StateCompleted {
				templateArg := m.templateArg
				if m.version != "" {
					if idx := strings.LastIndex(templateArg, "@"); idx != -1 {
						templateArg = templateArg[:idx] + "@" + m.version
					} else {
						templateArg = templateArg + "@" + m.version
					}
				}
				local := ""
				if m.useLocal {
					local = m.local
				}
				opts := install.Options{
					TemplateArg: templateArg,
					Dest:        m.dest,
					Session:     m.session,
					Local:       local,
				}
				err := install.Execute(context.Background(), opts)
				if err != nil {
					m.result = "Error: " + err.Error()
				} else {
					m.result = "Installed " + templateArg + " to " + m.dest
				}
				m.showForm = false
				return m, nil
			}
			if m.form.State == huh.StateAborted {
				m.showForm = false
				m.result = "Cancelled"
				return m, nil
			}
		}
		return m, cmd
	}
	return m, nil
}

func (m *DocsCreateModel) buildForm() {
	if len(m.templates) == 0 {
		return
	}
	opts := make([]huh.Option[string], len(m.templates))
	for i, t := range m.templates {
		opts[i] = huh.NewOption(t.Name+" - "+t.Description, t.Name)
	}
	var sessionField huh.Field
	if m.project != nil && m.project.IsProject && m.project.Config.Mode == "multi" {
		sOpts := make([]huh.Option[string], len(m.project.Config.Sessions))
		for i, s := range m.project.Config.Sessions {
			sOpts[i] = huh.NewOption(s, s)
		}
		sessionField = huh.NewSelect[string]().Title("Session").Options(sOpts...).Value(&m.session)
	} else {
		sessionField = huh.NewInput().Title("Session (only for multi)").Value(&m.session).Placeholder("leave empty for single")
	}
	m.templateArg = m.templates[0].Name
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Template").Options(opts...).Value(&m.templateArg),
			huh.NewInput().Title("Version (semver constraint or empty=latest)").Placeholder("^1.0.0").Value(&m.version).Validate(func(s string) error {
				if s == "" {
					return nil
				}
				if strings.Contains(s, " ") {
					return fmt.Errorf("invalid version")
				}
				return nil
			}),
			huh.NewInput().Title("Destination").Value(&m.dest),
			huh.NewConfirm().Title("Use local directory?").Value(&m.useLocal),
			huh.NewInput().Title("Local path (if use local)").Value(&m.local),
			sessionField,
		),
	)
	if m.project != nil && m.project.IsProject {
		m.dest = m.project.Root
	}
}

func (m DocsCreateModel) View() string {
	if m.loading {
		return lipgloss.NewStyle().Padding(1).Render(m.spinner.View()+" Loading templates...")
	}
	if m.err != "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Padding(1).Render(fmt.Sprintf("Error: %s\nPress r to retry", m.err))
	}
	if m.result != "" {
		return lipgloss.NewStyle().Padding(1).Render(m.result + "\n\nPress esc to back")
	}
	if m.showForm && m.form != nil {
		return lipgloss.NewStyle().Padding(1).Render(m.form.View())
	}
	return lipgloss.NewStyle().Padding(1).Render("No templates available")
}

func (m *DocsCreateModel) SetProject(p *ProjectContext) { m.project = p }
