package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/UNSAReport/tui/internal/config"
	"github.com/UNSAReport/tui/internal/naming"
)

type PrepareModel struct {
	project *ProjectContext
	vars    map[string]string
	fileTemplate string
	submissionDir string
	srcDir string
	reportFile string
	reportWord string
	codeWord string
	form *huh.Form
	showForm bool
	result string
}

func NewPrepareModel(project *ProjectContext) PrepareModel {
	return PrepareModel{project: project}
}

func (m PrepareModel) Init() tea.Cmd {
	return m.loadContext()
}

func (m PrepareModel) loadContext() tea.Cmd {
	return func() tea.Msg {
		return prepareLoadedMsg{}
	}
}

type prepareLoadedMsg struct{}

func (m PrepareModel) Update(msg tea.Msg) (PrepareModel, tea.Cmd) {
	switch msg.(type) {
	case prepareLoadedMsg:
		// Load defaults from project config
		if m.project != nil && m.project.IsProject {
			cfg := m.project.Config
			m.fileTemplate = cfg.Prepare.Output.FileTemplate
			if m.fileTemplate == "" {
				m.fileTemplate = "{output_type}_{lab_number}"
			}
			m.submissionDir = cfg.Prepare.Output.SubmissionDir
			if m.submissionDir == "" {
				m.submissionDir = "submission"
			}
			m.srcDir = cfg.Prepare.Input.SrcDir
			if m.srcDir == "" {
				m.srcDir = "src"
			}
			m.reportFile = cfg.Prepare.Input.ReportFile
			if m.reportFile == "" {
				m.reportFile = "report.typ"
			}
			m.reportWord = cfg.Prepare.Output.ReportWord
			if m.reportWord == "" {
				m.reportWord = "Informe"
			}
			m.codeWord = cfg.Prepare.Output.CodeWord
			if m.codeWord == "" {
				m.codeWord = "Código Fuente"
			}
			// vars: try to read report.typ metadata? For now stub with example vars
			m.vars = map[string]string{
				"lab_number": "01",
				"course": "Curso",
			}
		} else {
			m.fileTemplate = "{output_type}_{lab_number}"
			m.submissionDir = "submission"
			m.srcDir = "src"
			m.reportFile = "report.typ"
			m.reportWord = "Informe"
			m.codeWord = "Código Fuente"
			m.vars = map[string]string{"lab_number": "01"}
		}
		m.buildForm()
		m.showForm = true
		if m.form != nil {
			return m, m.form.Init()
		}
		return m, nil
	case tea.KeyMsg:
		// handled via form
	}
	if m.showForm && m.form != nil {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
			if m.form.State == huh.StateCompleted {
				// Save config
				if m.project != nil && m.project.IsProject {
					cfg := m.project.Config
					cfg.Prepare.Output.FileTemplate = m.fileTemplate
					cfg.Prepare.Output.SubmissionDir = m.submissionDir
					cfg.Prepare.Input.SrcDir = m.srcDir
					cfg.Prepare.Input.ReportFile = m.reportFile
					cfg.Prepare.Output.ReportWord = m.reportWord
					cfg.Prepare.Output.CodeWord = m.codeWord
					_ = config.WriteConfig(m.project.Root, cfg)
					m.result = "Configuration saved"
				} else {
					m.result = "Preview: " + naming.ApplyTemplate(m.fileTemplate, m.vars, m.reportWord) + ".pdf"
				}
				m.showForm = false
				return m, nil
			}
			if m.form.State == huh.StateAborted {
				m.showForm = false
				return m, nil
			}
		}
		return m, cmd
	}
	return m, nil
}

func (m *PrepareModel) buildForm() {
	// preview not needed as form title will show live?
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("File template").Value(&m.fileTemplate).Placeholder("{output_type}_{lab_number}"),
			huh.NewInput().Title("Submission dir").Value(&m.submissionDir),
			huh.NewInput().Title("Source dir").Value(&m.srcDir),
			huh.NewInput().Title("Report file").Value(&m.reportFile),
			huh.NewInput().Title("Report word").Value(&m.reportWord),
			huh.NewInput().Title("Code word").Value(&m.codeWord),
		),
	)
}

func (m PrepareModel) View() string {
	if m.result != "" {
		previewReport := naming.ApplyTemplate(m.fileTemplate, m.vars, m.reportWord)
		previewCode := naming.ApplyTemplate(m.fileTemplate, m.vars, m.codeWord)
		return lipgloss.NewStyle().Padding(1).Render(fmt.Sprintf("Result: %s\n\nPreview: %s.pdf | %s.zip\n\nPress esc to back", m.result, previewReport, previewCode))
	}
	if m.showForm && m.form != nil {
		previewReport := naming.ApplyTemplate(m.fileTemplate, m.vars, m.reportWord)
		previewCode := naming.ApplyTemplate(m.fileTemplate, m.vars, m.codeWord)
		preview := fmt.Sprintf("Preview: %s.pdf | %s.zip", previewReport, previewCode)
		// Show form plus preview
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Padding(1).Render(m.form.View()),
			lipgloss.NewStyle().Padding(0,1).Foreground(lipgloss.Color("#888")).Render(preview),
		)
	}
	return lipgloss.NewStyle().Padding(1).Render("Loading prepare context...")
}

func (m *PrepareModel) SetProject(p *ProjectContext) { m.project = p }
