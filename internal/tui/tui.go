package tui

import (
	"fmt"
	"strings"

	"hosts-manager/internal/config"
	"hosts-manager/internal/hosts"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	hostsFile    *hosts.HostsFile
	config       *config.Config
	currentView  view
	cursor       int
	selected     map[int]bool
	searchQuery  string
	searchActive bool
	message      string
	entries      []entryWithIndex
	categories   []string
	width        int
	height       int
}

type view int

const (
	viewMain view = iota
	viewSearch
	viewHelp
)

type entryWithIndex struct {
	entry    hosts.Entry
	category string
	index    int
	catIndex int
}

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Margin(1, 0, 0, 2)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Bold(true).
			Margin(0, 0, 1, 2)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)

	enabledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("76"))

	disabledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	categoryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true).
			Margin(1, 0, 0, 0)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Margin(1, 0, 0, 2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("76")).
			Bold(true)
)

func Run(hostsFile *hosts.HostsFile, cfg *config.Config) error {
	m := model{
		hostsFile:   hostsFile,
		config:      cfg,
		currentView: viewMain,
		selected:    make(map[int]bool),
		entries:     buildEntryList(hostsFile),
	}

	m.categories = make([]string, len(hostsFile.Categories))
	for i, cat := range hostsFile.Categories {
		m.categories[i] = cat.Name
	}

	p := tea.NewProgram(&m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func buildEntryList(hostsFile *hosts.HostsFile) []entryWithIndex {
	var entries []entryWithIndex
	index := 0

	for catIndex, category := range hostsFile.Categories {
		for _, entry := range category.Entries {
			entries = append(entries, entryWithIndex{
				entry:    entry,
				category: category.Name,
				index:    index,
				catIndex: catIndex,
			})
			index++
		}
	}

	return entries
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.currentView {
		case viewMain:
			return m.updateMain(msg)
		case viewSearch:
			return m.updateSearch(msg)
		case viewHelp:
			return m.updateHelp(msg)
		}
	}

	return m, nil
}

func (m *model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}

	case "g":
		m.cursor = 0

	case "G":
		m.cursor = len(m.entries) - 1

	case "space":
		if m.cursor < len(m.entries) {
			entry := &m.entries[m.cursor]
			entry.entry.Enabled = !entry.entry.Enabled

			hostsCategory := m.hostsFile.GetCategory(entry.category)
			if hostsCategory != nil {
				for i := range hostsCategory.Entries {
					if hostsCategory.Entries[i].IP == entry.entry.IP &&
						strings.Join(hostsCategory.Entries[i].Hostnames, " ") == strings.Join(entry.entry.Hostnames, " ") {
						hostsCategory.Entries[i].Enabled = entry.entry.Enabled
						break
					}
				}
			}

			status := "disabled"
			if entry.entry.Enabled {
				status = "enabled"
			}
			m.message = fmt.Sprintf("Entry %s", status)
		}

	case "d":
		if m.cursor < len(m.entries) {
			entry := m.entries[m.cursor]
			hostname := entry.entry.Hostnames[0]

			if m.hostsFile.RemoveEntry(hostname) {
				m.entries = buildEntryList(m.hostsFile)
				if m.cursor >= len(m.entries) && len(m.entries) > 0 {
					m.cursor = len(m.entries) - 1
				}
				m.message = fmt.Sprintf("Deleted entry: %s", hostname)
			} else {
				m.message = fmt.Sprintf("Failed to delete entry: %s", hostname)
			}
		}

	case "/":
		m.currentView = viewSearch
		m.searchActive = true
		m.searchQuery = ""

	case "r":
		m.entries = buildEntryList(m.hostsFile)
		m.message = "Refreshed"

	case "s":
		return m, m.saveFile()

	case "?", "h":
		m.currentView = viewHelp

	case "enter":
		if m.cursor < len(m.entries) {
			entry := m.entries[m.cursor]
			m.message = fmt.Sprintf("Selected: %s -> %v", entry.entry.IP, entry.entry.Hostnames)
		}
	}

	return m, nil
}

func (m *model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentView = viewMain
		m.searchActive = false
		m.searchQuery = ""
		m.entries = buildEntryList(m.hostsFile)

	case "enter":
		m.currentView = viewMain
		m.searchActive = false
		m.filterEntries()

	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}

	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
		}
	}

	return m, nil
}

func (m *model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "?", "h":
		m.currentView = viewMain
	}

	return m, nil
}

func (m *model) filterEntries() {
	if m.searchQuery == "" {
		m.entries = buildEntryList(m.hostsFile)
		return
	}

	var filtered []entryWithIndex
	query := strings.ToLower(m.searchQuery)

	for _, entry := range m.entries {
		match := false

		for _, hostname := range entry.entry.Hostnames {
			if strings.Contains(strings.ToLower(hostname), query) {
				match = true
				break
			}
		}

		if !match && strings.Contains(strings.ToLower(entry.entry.IP), query) {
			match = true
		}

		if !match && strings.Contains(strings.ToLower(entry.entry.Comment), query) {
			match = true
		}

		if !match && strings.Contains(strings.ToLower(entry.category), query) {
			match = true
		}

		if match {
			filtered = append(filtered, entry)
		}
	}

	m.entries = filtered
	m.cursor = 0
	m.message = fmt.Sprintf("Found %d entries matching '%s'", len(filtered), m.searchQuery)
}

func (m *model) saveFile() tea.Cmd {
	return func() tea.Msg {
		if err := m.hostsFile.Write(m.hostsFile.FilePath); err != nil {
			return errorMsg{err}
		}
		return successMsg{}
	}
}

type errorMsg struct{ err error }
type successMsg struct{}

func (m *model) View() string {
	switch m.currentView {
	case viewMain:
		return m.viewMain()
	case viewSearch:
		return m.viewSearch()
	case viewHelp:
		return m.viewHelp()
	}

	return ""
}

func (m *model) viewMain() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Hosts Manager"))
	b.WriteString("\n")

	if m.searchQuery != "" {
		b.WriteString(headerStyle.Render(fmt.Sprintf("Search: %s (%d results)", m.searchQuery, len(m.entries))))
	} else {
		b.WriteString(headerStyle.Render(fmt.Sprintf("Total entries: %d", len(m.entries))))
	}

	currentCategory := ""
	for i, entry := range m.entries {
		if entry.category != currentCategory {
			currentCategory = entry.category
			b.WriteString(categoryStyle.Render(fmt.Sprintf("\n=== %s ===", strings.ToUpper(currentCategory))))
			b.WriteString("\n")
		}

		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}

		status := "✗"
		style := disabledStyle
		if entry.entry.Enabled {
			status = "✓"
			style = enabledStyle
		}

		line := fmt.Sprintf("%s%s %s -> %s",
			cursor,
			status,
			entry.entry.IP,
			strings.Join(entry.entry.Hostnames, " "))

		if entry.entry.Comment != "" {
			line += " # " + entry.entry.Comment
		}

		if m.cursor == i {
			line = selectedStyle.Render(line)
		} else {
			line = style.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.message != "" {
		b.WriteString("\n")
		if strings.HasPrefix(m.message, "Error") || strings.HasPrefix(m.message, "Failed") {
			b.WriteString(errorStyle.Render(m.message))
		} else {
			b.WriteString(successStyle.Render(m.message))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("\nControls: space=toggle, d=delete, s=save, /=search, ?=help, q=quit"))

	return b.String()
}

func (m *model) viewSearch() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Search Mode"))
	b.WriteString("\n")

	b.WriteString("Enter search query: ")
	b.WriteString(m.searchQuery)
	b.WriteString("_")
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("Press Enter to search, Esc to cancel"))

	return b.String()
}

func (m *model) viewHelp() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Help"))
	b.WriteString("\n")

	helpText := `
Navigation:
  ↑/k       Move cursor up
  ↓/j       Move cursor down
  g         Go to top
  G         Go to bottom

Actions:
  space     Toggle entry enabled/disabled
  d         Delete entry
  s         Save changes to hosts file
  r         Refresh entry list
  /         Search entries
  enter     Show entry details

Views:
  ?/h       Show/hide this help
  q         Quit application
  esc       Cancel current action

Search:
  Search works on hostnames, IPs, comments, and categories.
  Press Enter to apply search, Esc to cancel.
`

	b.WriteString(helpText)
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press ? or h to return to main view"))

	return b.String()
}