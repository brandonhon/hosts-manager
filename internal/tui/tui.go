package tui

import (
	"fmt"
	"strings"

	"hosts-manager/internal/config"
	"hosts-manager/internal/hosts"

	tea "github.com/charmbracelet/bubbletea"
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
	// Add entry fields
	addIP        string
	addHostnames string
	addComment   string
	addCategory  string
	addField     int // 0=IP, 1=hostnames, 2=comment, 3=category
	// Move entry fields
	moveEntryIndex     int    // Index of entry to move
	moveCategoryCursor int    // Cursor for category selection
	moveTargetCategory string // Target category name
	// Create category fields
	createCategoryName        string // Name of new category to create
	createCategoryDescription string // Description of new category
	createCategoryField       int    // 0=name, 1=description
}

type view int

const (
	viewMain view = iota
	viewSearch
	viewHelp
	viewAdd
	viewMove
	viewCreateCategory
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

	moveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Background(lipgloss.Color("53")).
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
		case viewAdd:
			return m.updateAdd(msg)
		case viewMove:
			return m.updateMove(msg)
		case viewCreateCategory:
			return m.updateCreateCategory(msg)
		}

	case errorMsg:
		m.message = fmt.Sprintf("Error: %v", msg.err)
		return m, nil

	case successMsg:
		m.message = "File saved successfully!"
		return m, nil
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

	case " ":
		if m.cursor < len(m.entries) {
			entry := &m.entries[m.cursor]
			entry.entry.Enabled = !entry.entry.Enabled

			// Update the corresponding entry in the hosts file
			hostsCategory := m.hostsFile.GetCategory(entry.category)
			if hostsCategory != nil {
				for i := range hostsCategory.Entries {
					// Match by IP and first hostname for more reliable identification
					if hostsCategory.Entries[i].IP == entry.entry.IP &&
						len(hostsCategory.Entries[i].Hostnames) > 0 &&
						len(entry.entry.Hostnames) > 0 &&
						hostsCategory.Entries[i].Hostnames[0] == entry.entry.Hostnames[0] {
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

	case "a":
		m.currentView = viewAdd
		m.addIP = ""
		m.addHostnames = ""
		m.addComment = ""
		m.addCategory = m.config.General.DefaultCategory
		m.addField = 0

	case "c":
		m.currentView = viewCreateCategory
		m.createCategoryName = ""
		m.createCategoryDescription = ""
		m.createCategoryField = 0

	case "m":
		if m.cursor < len(m.entries) {
			m.currentView = viewMove
			m.moveEntryIndex = m.cursor
			m.moveCategoryCursor = 0
			// Set initial target category to first available category different from current
			currentCategory := m.entries[m.cursor].category
			m.moveTargetCategory = ""
			for _, cat := range m.categories {
				if cat != currentCategory {
					m.moveTargetCategory = cat
					break
				}
			}
		} else {
			m.message = "No entry selected to move"
		}

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

func (m *model) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentView = viewMain

	case "tab", "down":
		m.addField = (m.addField + 1) % 4

	case "shift+tab", "up":
		m.addField = (m.addField + 3) % 4

	case "enter":
		if m.addIP != "" && m.addHostnames != "" {
			// Create new entry
			hostnames := strings.Fields(m.addHostnames)
			entry := hosts.Entry{
				IP:        m.addIP,
				Hostnames: hostnames,
				Comment:   m.addComment,
				Category:  m.addCategory,
				Enabled:   true,
			}

			// Add to hosts file
			if err := m.hostsFile.AddEntry(entry); err != nil {
				m.message = fmt.Sprintf("Error adding entry: %v", err)
				return m, nil
			}
			m.entries = buildEntryList(m.hostsFile)
			m.message = fmt.Sprintf("Added entry: %s -> %v", entry.IP, entry.Hostnames)
			m.currentView = viewMain
		} else {
			m.message = "IP and hostnames are required"
		}

	case "backspace":
		switch m.addField {
		case 0:
			if len(m.addIP) > 0 {
				m.addIP = m.addIP[:len(m.addIP)-1]
			}
		case 1:
			if len(m.addHostnames) > 0 {
				m.addHostnames = m.addHostnames[:len(m.addHostnames)-1]
			}
		case 2:
			if len(m.addComment) > 0 {
				m.addComment = m.addComment[:len(m.addComment)-1]
			}
		case 3:
			if len(m.addCategory) > 0 {
				m.addCategory = m.addCategory[:len(m.addCategory)-1]
			}
		}

	default:
		if len(msg.String()) == 1 {
			switch m.addField {
			case 0:
				m.addIP += msg.String()
			case 1:
				m.addHostnames += msg.String()
			case 2:
				m.addComment += msg.String()
			case 3:
				m.addCategory += msg.String()
			}
		}
	}

	return m, nil
}

func (m *model) updateMove(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentView = viewMain

	case "up", "k":
		if m.moveCategoryCursor > 0 {
			m.moveCategoryCursor--
			m.moveTargetCategory = m.getAvailableCategories()[m.moveCategoryCursor]
		}

	case "down", "j":
		availableCategories := m.getAvailableCategories()
		if m.moveCategoryCursor < len(availableCategories)-1 {
			m.moveCategoryCursor++
			m.moveTargetCategory = availableCategories[m.moveCategoryCursor]
		}

	case "enter":
		if m.moveTargetCategory != "" && m.moveEntryIndex < len(m.entries) {
			if err := m.moveEntry(m.moveEntryIndex, m.moveTargetCategory); err != nil {
				m.message = fmt.Sprintf("Error moving entry: %v", err)
			} else {
				entryToMove := m.entries[m.moveEntryIndex]
				m.message = fmt.Sprintf("Moved %s from %s to %s",
					entryToMove.entry.Hostnames[0],
					entryToMove.category,
					m.moveTargetCategory)
				m.entries = buildEntryList(m.hostsFile)
				// Try to keep cursor on the same entry after move
				m.cursor = m.findEntryAfterMove(entryToMove, m.moveTargetCategory)
			}
			m.currentView = viewMain
		}
	}

	return m, nil
}

func (m *model) updateCreateCategory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentView = viewMain

	case "tab", "down":
		m.createCategoryField = (m.createCategoryField + 1) % 2

	case "shift+tab", "up":
		m.createCategoryField = (m.createCategoryField + 1) % 2

	case "enter":
		if m.createCategoryName != "" {
			// Validate category name
			if err := m.validateCategoryName(m.createCategoryName); err != nil {
				m.message = fmt.Sprintf("Invalid category name: %v", err)
				return m, nil
			}

			// Check if category already exists
			for _, existingCategory := range m.categories {
				if existingCategory == m.createCategoryName {
					m.message = fmt.Sprintf("Category '%s' already exists", m.createCategoryName)
					return m, nil
				}
			}

			// Create new category
			if err := m.createCategory(m.createCategoryName, m.createCategoryDescription); err != nil {
				m.message = fmt.Sprintf("Error creating category: %v", err)
				return m, nil
			}

			// Update categories list and entries
			m.categories = append(m.categories, m.createCategoryName)
			m.entries = buildEntryList(m.hostsFile)
			m.message = fmt.Sprintf("Created category: %s", m.createCategoryName)
			m.currentView = viewMain
		} else {
			m.message = "Category name is required"
		}

	case "backspace":
		switch m.createCategoryField {
		case 0:
			if len(m.createCategoryName) > 0 {
				m.createCategoryName = m.createCategoryName[:len(m.createCategoryName)-1]
			}
		case 1:
			if len(m.createCategoryDescription) > 0 {
				m.createCategoryDescription = m.createCategoryDescription[:len(m.createCategoryDescription)-1]
			}
		}

	default:
		if len(msg.String()) == 1 {
			switch m.createCategoryField {
			case 0:
				m.createCategoryName += msg.String()
			case 1:
				m.createCategoryDescription += msg.String()
			}
		}
	}

	return m, nil
}

// validateCategoryName validates a category name using the same rules as the config validator
func (m *model) validateCategoryName(name string) error {
	if name == "" {
		return fmt.Errorf("category name cannot be empty")
	}

	if len(name) > 50 {
		return fmt.Errorf("category name too long (max 50 characters)")
	}

	// Use same validation as config validator
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return fmt.Errorf("category name contains invalid characters (only a-z, A-Z, 0-9, _, - allowed)")
		}
	}

	return nil
}

// createCategory adds a new category to the hosts file
func (m *model) createCategory(name, description string) error {
	// Create new category
	newCategory := hosts.Category{
		Name:        name,
		Description: description,
		Enabled:     true,
		Entries:     []hosts.Entry{},
	}

	// Add to hosts file
	m.hostsFile.Categories = append(m.hostsFile.Categories, newCategory)

	return nil
}

// getAvailableCategories returns categories excluding the current entry's category
func (m *model) getAvailableCategories() []string {
	if m.moveEntryIndex >= len(m.entries) {
		return m.categories
	}

	currentCategory := m.entries[m.moveEntryIndex].category
	var available []string
	for _, cat := range m.categories {
		if cat != currentCategory {
			available = append(available, cat)
		}
	}
	return available
}

// findEntryAfterMove tries to find the entry's new position after moving
func (m *model) findEntryAfterMove(movedEntry entryWithIndex, targetCategory string) int {
	for i, entry := range m.entries {
		if entry.category == targetCategory &&
			entry.entry.IP == movedEntry.entry.IP &&
			len(entry.entry.Hostnames) > 0 &&
			len(movedEntry.entry.Hostnames) > 0 &&
			entry.entry.Hostnames[0] == movedEntry.entry.Hostnames[0] {
			return i
		}
	}
	return 0 // Default to first entry if not found
}

// moveEntry moves an entry from its current category to the target category
func (m *model) moveEntry(entryIndex int, targetCategory string) error {
	if entryIndex >= len(m.entries) {
		return fmt.Errorf("invalid entry index")
	}

	entryToMove := m.entries[entryIndex]
	sourceCategory := entryToMove.category

	// Find the source category in hostsFile
	var sourceCat *hosts.Category
	for i := range m.hostsFile.Categories {
		if m.hostsFile.Categories[i].Name == sourceCategory {
			sourceCat = &m.hostsFile.Categories[i]
			break
		}
	}
	if sourceCat == nil {
		return fmt.Errorf("source category not found: %s", sourceCategory)
	}

	// Find the target category in hostsFile
	var targetCat *hosts.Category
	for i := range m.hostsFile.Categories {
		if m.hostsFile.Categories[i].Name == targetCategory {
			targetCat = &m.hostsFile.Categories[i]
			break
		}
	}
	if targetCat == nil {
		return fmt.Errorf("target category not found: %s", targetCategory)
	}

	// Find and remove the entry from source category
	var entryToMoveData hosts.Entry
	entryFound := false
	for i, entry := range sourceCat.Entries {
		if entry.IP == entryToMove.entry.IP &&
			len(entry.Hostnames) > 0 &&
			len(entryToMove.entry.Hostnames) > 0 &&
			entry.Hostnames[0] == entryToMove.entry.Hostnames[0] {
			entryToMoveData = entry
			entryToMoveData.Category = targetCategory
			// Remove from source category
			sourceCat.Entries = append(sourceCat.Entries[:i], sourceCat.Entries[i+1:]...)
			entryFound = true
			break
		}
	}

	if !entryFound {
		return fmt.Errorf("entry not found in source category")
	}

	// Add to target category
	targetCat.Entries = append(targetCat.Entries, entryToMoveData)

	return nil
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
	case viewAdd:
		return m.viewAdd()
	case viewMove:
		return m.viewMove()
	case viewCreateCategory:
		return m.viewCreateCategory()
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

	b.WriteString(helpStyle.Render("\nControls: space=toggle, a=add, c=create category, m=move, d=delete, s=save, /=search, ?=help, q=quit"))

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
  a         Add new entry
  c         Create new category
  m         Move entry to different category
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

func (m *model) viewAdd() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Add New Entry"))
	b.WriteString("\n\n")

	// IP field
	ipLabel := "IP Address:"
	if m.addField == 0 {
		ipLabel = selectedStyle.Render("IP Address:")
	}
	b.WriteString(fmt.Sprintf("%s %s\n", ipLabel, m.addIP))

	// Hostnames field
	hostnamesLabel := "Hostnames:"
	if m.addField == 1 {
		hostnamesLabel = selectedStyle.Render("Hostnames:")
	}
	b.WriteString(fmt.Sprintf("%s %s\n", hostnamesLabel, m.addHostnames))

	// Comment field
	commentLabel := "Comment (optional):"
	if m.addField == 2 {
		commentLabel = selectedStyle.Render("Comment (optional):")
	}
	b.WriteString(fmt.Sprintf("%s %s\n", commentLabel, m.addComment))

	// Category field
	categoryLabel := "Category:"
	if m.addField == 3 {
		categoryLabel = selectedStyle.Render("Category:")
	}
	b.WriteString(fmt.Sprintf("%s %s\n", categoryLabel, m.addCategory))

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Use Tab/Shift+Tab to navigate fields"))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press Enter to add entry, Esc to cancel"))

	return b.String()
}

func (m *model) viewMove() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Move Entry to Category"))
	b.WriteString("\n\n")

	// Show the entry being moved
	if m.moveEntryIndex < len(m.entries) {
		entry := m.entries[m.moveEntryIndex]
		entryStr := fmt.Sprintf("Moving: %s -> %s",
			entry.entry.IP,
			strings.Join(entry.entry.Hostnames, " "))
		if entry.entry.Comment != "" {
			entryStr += " # " + entry.entry.Comment
		}
		b.WriteString(moveStyle.Render(entryStr))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("From category: %s\n\n", entry.category))
	}

	b.WriteString("Select target category:\n")

	// Show available categories
	availableCategories := m.getAvailableCategories()
	for i, category := range availableCategories {
		cursor := "  "
		if i == m.moveCategoryCursor {
			cursor = "> "
		}

		line := cursor + category
		if i == m.moveCategoryCursor {
			line = selectedStyle.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Use ↑/↓ to select category, Enter to move, Esc to cancel"))

	return b.String()
}

func (m *model) viewCreateCategory() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Create New Category"))
	b.WriteString("\n\n")

	// Name field
	nameLabel := "Category Name:"
	if m.createCategoryField == 0 {
		nameLabel = selectedStyle.Render("Category Name:")
	}
	b.WriteString(fmt.Sprintf("%s %s\n", nameLabel, m.createCategoryName))

	// Description field
	descLabel := "Description (optional):"
	if m.createCategoryField == 1 {
		descLabel = selectedStyle.Render("Description (optional):")
	}
	b.WriteString(fmt.Sprintf("%s %s\n", descLabel, m.createCategoryDescription))

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Category names can contain: a-z, A-Z, 0-9, _, -"))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Use Tab/Shift+Tab to navigate fields"))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press Enter to create category, Esc to cancel"))

	return b.String()
}
