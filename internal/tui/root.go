package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/UNSAReport/tui/internal/config"
	"github.com/UNSAReport/tui/internal/i18n"
)

type AppID int

const (
	AppRegistry AppID = iota
	AppDocs
	AppAuth
	AppSlides
)

func (a AppID) String() string {
	switch a {
	case AppRegistry:
		return "Registry"
	case AppDocs:
		return "Docs"
	case AppAuth:
		return "Auth"
	case AppSlides:
		return "Slides"
	default:
		return "Unknown"
	}
}

type Category struct {
	Name      string
	Functions []string
}

type AppDef struct {
	ID         AppID
	Name       string
	Categories []Category
}
func appDefs() []AppDef {
	return []AppDef{
		{ID: AppRegistry, Name: i18n.T("app.registry"), Categories: []Category{{Name: i18n.T("category.templates"), Functions: []string{i18n.T("function.browse"), i18n.T("function.search")}}}},
		{ID: AppDocs, Name: i18n.T("app.docs"), Categories: []Category{
			{Name: i18n.T("category.create"), Functions: []string{i18n.T("function.new_project")}},
			{Name: i18n.T("category.update"), Functions: []string{i18n.T("function.check"), i18n.T("function.apply"), i18n.T("function.rollback")}},
			{Name: i18n.T("category.prepare"), Functions: []string{i18n.T("function.configure"), i18n.T("function.build")}},
		}},
		{ID: AppAuth, Name: i18n.T("app.auth"), Categories: []Category{{Name: i18n.T("category.account"), Functions: []string{i18n.T("function.status"), i18n.T("function.login"), i18n.T("function.logout")}}}},
		{ID: AppSlides, Name: i18n.T("app.slides"), Categories: []Category{{Name: i18n.T("category.overview"), Functions: []string{i18n.T("coming_soon_slides")}}}},
	}
}

type ProjectContext = config.ProjectContext

type RootOptions struct {
	Project *ProjectContext
}

type navItem struct {
	Category string
	Function string
	App      AppID
}

type RootModel struct {
	activeApp     AppID
	apps          []AppDef
	nav           []navItem
	cursor        int
	sidebarFocus  bool
	width, height int
	showHelp      bool
	project       *ProjectContext
	styles        Styles
	keys          KeyMap
	help          help.Model
	registry      RegistryModel
	docsCreate    DocsCreateModel
	docsUpdate    DocsUpdateModel
	docsPrepare   PrepareModel
	auth          AuthModel
}
func NewRootModel(opts RootOptions) RootModel {
	i18n.Init()
	apps := appDefs()
	m := RootModel{
		activeApp:    AppRegistry,
		apps:         apps,
		sidebarFocus: true,
		styles:       newStyles(defaultTheme()),
		keys:         DefaultKeyMap(),
		help:         help.New(),
		project:      opts.Project,
		registry:     NewRegistryModel(),
		docsCreate:   NewDocsCreateModel(nil),
		docsUpdate:   NewDocsUpdateModel(),
		docsPrepare:  NewPrepareModel(nil),
		auth:         NewAuthModel(),
	}
	m.rebuildNav()
	if m.project == nil {
		cwd, _ := os.Getwd()
		if pc, err := config.DetectProject(cwd); err == nil {
			m.project = pc
		} else {
			m.project = &ProjectContext{IsProject: false}
		}
	}
	m.docsCreate.SetProject(m.project)
	m.docsPrepare.SetProject(m.project)
	return m
}
func (m *RootModel) rebuildNav() {
	m.nav = nil
	for _, a := range m.apps {
		if a.ID != m.activeApp {
			continue
		}
		for _, c := range a.Categories {
			for _, f := range c.Functions {
				m.nav = append(m.nav, navItem{Category: c.Name, Function: f, App: a.ID})
			}
		}
	}
	if m.cursor >= len(m.nav) {
		m.cursor = 0
	}
}

func (m RootModel) Init() tea.Cmd {
	return tea.Batch(
		m.registry.Init(),
		m.docsCreate.Init(),
		m.docsPrepare.Init(),
		m.auth.Init(),
	)
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.registry.SetSize(m.width, m.height)
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.registry, cmd = m.registry.Update(msg)
		cmds = append(cmds, cmd)
		m.docsCreate, cmd = m.docsCreate.Update(msg)
		cmds = append(cmds, cmd)
		m.docsUpdate, cmd = m.docsUpdate.Update(msg)
		cmds = append(cmds, cmd)
		m.docsPrepare, cmd = m.docsPrepare.Update(msg)
		cmds = append(cmds, cmd)
		m.auth, cmd = m.auth.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case docsCreateFetchedMsg, prepareLoadedMsg, authStatusMsg, registryFetchedMsg:
		var cmd tea.Cmd
		m.registry, cmd = m.registry.Update(msg)
		_ = cmd
		m.docsCreate, cmd = m.docsCreate.Update(msg)
		_ = cmd
		m.docsPrepare, cmd = m.docsPrepare.Update(msg)
		_ = cmd
		m.auth, cmd = m.auth.Update(msg)
		_ = cmd
		return m, cmd
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, m.keys.Tab):
			m.sidebarFocus = !m.sidebarFocus
			return m, nil
		case key.Matches(msg, m.keys.Back):
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			if !m.sidebarFocus {
				if m.activeApp == AppRegistry {
					var cmd tea.Cmd
					m.registry, cmd = m.registry.Update(msg)
					if m.registry.mode == registryBrowse {
					}
					return m, cmd
				}
				if m.activeApp == AppAuth {
					var cmd tea.Cmd
					m.auth, cmd = m.auth.Update(msg)
					return m, cmd
				}
				if m.activeApp == AppDocs {
					cur := navItem{}
					if len(m.nav) > 0 && m.cursor < len(m.nav) {
						cur = m.nav[m.cursor]
					}
					if cur.Category == i18n.T("category.create") {
						var cmd tea.Cmd
						m.docsCreate, cmd = m.docsCreate.Update(msg)
						return m, cmd
					}
					if cur.Category == i18n.T("category.prepare") {
						var cmd tea.Cmd
						m.docsPrepare, cmd = m.docsPrepare.Update(msg)
						return m, cmd
					}
				}
			}
			m.sidebarFocus = true
			return m, nil
		}
		switch msg.String() {
		case "1":
			m.activeApp = AppRegistry
			m.cursor = 0
			m.rebuildNav()
			return m, nil
		case "2":
			m.activeApp = AppDocs
			m.cursor = 0
			m.rebuildNav()
			return m, nil
		case "3":
			m.activeApp = AppAuth
			m.cursor = 0
			m.rebuildNav()
			return m, nil
		case "4":
			m.activeApp = AppSlides
			m.cursor = 0
			m.rebuildNav()
			return m, nil
		case "h", "left":
			m.activeApp = (m.activeApp - 1 + 4) % 4
			m.cursor = 0
			m.rebuildNav()
			return m, nil
		case "l", "right":
			m.activeApp = (m.activeApp + 1) % 4
			m.cursor = 0
			m.rebuildNav()
			return m, nil
		}
		if !m.sidebarFocus {
			if m.activeApp == AppRegistry {
				var cmd tea.Cmd
				m.registry, cmd = m.registry.Update(msg)
				return m, cmd
			}
			if m.activeApp == AppDocs {
				cur := navItem{}
				if len(m.nav) > 0 && m.cursor < len(m.nav) {
					cur = m.nav[m.cursor]
				}
				if cur.Category == i18n.T("category.create") {
					var cmd tea.Cmd
					m.docsCreate, cmd = m.docsCreate.Update(msg)
					return m, cmd
				}
				if cur.Category == i18n.T("category.update") {
					var cmd tea.Cmd
					m.docsUpdate, cmd = m.docsUpdate.Update(msg)
					return m, cmd
				}
				if cur.Category == i18n.T("category.prepare") {
					var cmd tea.Cmd
					m.docsPrepare, cmd = m.docsPrepare.Update(msg)
					return m, cmd
				}
			}
			if m.activeApp == AppAuth {
				var cmd tea.Cmd
				m.auth, cmd = m.auth.Update(msg)
				return m, cmd
			}
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			if len(m.nav) > 0 && m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if len(m.nav) > 0 && m.cursor < len(m.nav)-1 {
				m.cursor++
			}
			return m, nil
		case "enter":
			if len(m.nav) > 0 {
				cur := m.nav[m.cursor]
				if m.activeApp == AppRegistry {
					if cur.Function == i18n.T("function.search") {
						m.registry.mode = registrySearch
						m.registry.focusSearch = true
						m.registry.textInput.Focus()
					} else {
						m.registry.mode = registryBrowse
					}
				}
				if m.activeApp == AppAuth {
					if cur.Function == i18n.T("function.login") {
						m.auth.SetMode(authLogin)
					} else if cur.Function == i18n.T("function.logout") {
						m.auth.SetMode(authLogout)
					} else {
						m.auth.SetMode(authStatus)
					}
				}
				m.sidebarFocus = false
			}
			return m, nil
		case "r":
			if m.activeApp == AppRegistry {
				var cmd tea.Cmd
				m.registry, cmd = m.registry.Update(msg)
				return m, cmd
			}
			if m.activeApp == AppAuth {
				var cmd tea.Cmd
				m.auth, cmd = m.auth.Update(msg)
				return m, cmd
			}
			return m, nil
		}
	}
	if m.activeApp == AppRegistry {
		var cmd tea.Cmd
		m.registry, cmd = m.registry.Update(msg)
		return m, cmd
	}
	if m.activeApp == AppAuth {
		var cmd tea.Cmd
		m.auth, cmd = m.auth.Update(msg)
		return m, cmd
	}
	if m.activeApp == AppDocs {
		var cmd tea.Cmd
		m.docsCreate, cmd = m.docsCreate.Update(msg)
		if cmd != nil {
			return m, cmd
		}
		m.docsPrepare, cmd = m.docsPrepare.Update(msg)
		if cmd != nil {
			return m, cmd
		}
		m.docsUpdate, cmd = m.docsUpdate.Update(msg)
		return m, cmd
	}
	return m, nil
}
func (m RootModel) View() string {
	if m.width == 0 {
		m.width = 80
	}
	if m.height == 0 {
		m.height = 24
	}

	var tabs []string
	for _, a := range m.apps {
		name := fmt.Sprintf(" %d:%s ", int(a.ID)+1, a.Name)
		if a.ID == m.activeApp {
			tabs = append(tabs, m.styles.TopBarActive.Render(name))
		} else {
			tabs = append(tabs, m.styles.TopBar.Render(name))
		}
	}
	var badge string
	if m.project != nil && m.project.IsProject {
		label := m.project.Root
		if len(label) > 20 {
			parts := strings.Split(label, "/")
			label = parts[len(parts)-1]
		}
		tpl := m.project.Config.Template
		mode := m.project.Config.Mode
		badge = m.styles.TopBarMuted.Render(fmt.Sprintf(" %s [%s %s] ", label, tpl, mode))
	} else {
		badge = m.styles.TopBarMuted.Render(" " + i18n.T("no_project") + " ")
	}
	topBar := lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(tabs, ""), badge)
	topBar = lipgloss.PlaceHorizontal(m.width, lipgloss.Left, topBar)

	availH := m.height - lipgloss.Height(topBar) - 1
	if availH < 5 {
		availH = 5
	}
	sidebarW := m.width / 4
	if sidebarW < 24 {
		sidebarW = 24
	}
	if sidebarW > 40 {
		sidebarW = 40
	}
	mainW := m.width - sidebarW - 2
	if mainW < 30 {
		mainW = 30
	}

	var sb strings.Builder
	activeApp := m.apps[m.activeApp]
	for _, cat := range activeApp.Categories {
		sb.WriteString(m.styles.SidebarSel.Render(cat.Name) + "\n")
		for _, fn := range cat.Functions {
			idx := -1
			for i, n := range m.nav {
				if n.Category == cat.Name && n.Function == fn {
					idx = i
					break
				}
			}
			prefix := "  "
			style := lipgloss.NewStyle()
			if idx == m.cursor {
				prefix = "> "
				style = m.styles.SidebarSel
				if m.sidebarFocus {
					style = style.Background(lipgloss.Color("#313244"))
				}
			}
			sb.WriteString(style.Render(prefix+fn) + "\n")
		}
	}
	sidebarContent := sb.String()
	sidebarBox := m.styles.Sidebar.Width(sidebarW).Height(availH).Render(sidebarContent)

	var mainContent string
	if m.activeApp == AppRegistry {
		m.registry.SetSize(mainW, availH)
		mainContent = m.registry.View()
	} else if m.activeApp == AppDocs && len(m.nav) > 0 && m.cursor < len(m.nav) {
		cur := m.nav[m.cursor]
		switch cur.Category {
		case i18n.T("category.create"):
			mainContent = m.docsCreate.View()
		case i18n.T("category.update"):
			m.docsUpdate.SetSize(mainW, availH)
			mainContent = m.docsUpdate.View()
		case i18n.T("category.prepare"):
			mainContent = m.docsPrepare.View()
		default:
			mainContent = m.renderMain(cur, mainW, availH)
		}
	} else if m.activeApp == AppAuth {
		mainContent = m.auth.View()
	} else if len(m.nav) > 0 && m.cursor < len(m.nav) {
		cur := m.nav[m.cursor]
		mainContent = m.renderMain(cur, mainW, availH)
	} else {
		mainContent = "No function selected"
	}
	mainBox := m.styles.Main.Width(mainW).Height(availH).Render(mainContent)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, mainBox)

	helpBar := m.styles.HelpBar.Render(i18n.T("help.bar"))
	if m.showHelp {
		helpView := m.help.View(m.keys)
		overlay := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.styles.Theme.Border).
			Padding(1, 2).
			Width(min(m.width-4, 60)).
			Render("Help\n\n" + helpView + "\n\nPress ? or esc to close")
		body = lipgloss.Place(m.width, availH, lipgloss.Center, lipgloss.Center, overlay, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("0")))
		return lipgloss.JoinVertical(lipgloss.Left, topBar, body, helpBar)
	}

	return lipgloss.JoinVertical(lipgloss.Left, topBar, body, helpBar)
}

func (m RootModel) renderMain(cur navItem, w, h int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(cur.Category+" > "+cur.Function) + "\n\n")

	switch m.activeApp {
	case AppRegistry:
		switch cur.Function {
		case "Browse":
			b.WriteString("Registry Browse — spinner then list populates.\n")
			b.WriteString("(Phase 2: registry client + bubbles/list)\n")
		case "Search":
			b.WriteString("Search — filter templates in-memory (Phase 2)\n")
		}
	case AppDocs:
		switch cur.Category {
		case "Create":
			b.WriteString("Docs > Create — template install form (Phase 4)\n")
		case "Update":
			b.WriteString("Docs > Update — diff/prompt/rollback (Phase 4)\n")
		case "Prepare":
			b.WriteString("Docs > Prepare — configure + build (Phase 4)\n")
		}
	case AppAuth:
		b.WriteString("Auth — Status/Login/Logout (Phase 3)\n")
		b.WriteString("Endpoint + token indicator\n")
	case AppSlides:
		if cur.Function == "Coming soon — slides management" {
			b.WriteString("Coming soon — slides management\n")
		}
	}
	content := b.String()
	lines := strings.Split(content, "\n")
	if len(lines) > h-2 {
		lines = lines[:h-2]
		content = strings.Join(lines, "\n")
	}
	return content
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
