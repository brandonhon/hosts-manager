// Package tui is the interactive hosts file editor: categories on the left,
// the selected category's entries on the right. Edits stay in memory until
// saved; saving goes through the same backup-write-audit path as the CLI.
package tui

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/brandonhon/hosts-manager/internal/config"
	"github.com/brandonhon/hosts-manager/internal/hosts"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Below this width the category pane is hidden and ←/→ switch category.
const narrowWidth = 80

const sidebarWidth = 26

type focus int

const (
	focusCategories focus = iota
	focusEntries
)

type mode int

const (
	modeBrowse mode = iota
	modeFilter
	modeForm
	modePick
	modeHelp
	modeConfirmQuit
)

// row identifies an entry by its position in the hosts file. Positions, not
// IP/hostname, so that duplicate lines are still distinct rows.
type row struct{ cat, idx int }

type model struct {
	hf     *hosts.HostsFile
	cfg    *config.Config
	dryRun bool
	save   func(*hosts.HostsFile) error

	width, height int
	focus         focus
	mode          mode

	cat    int // selected category index
	cursor int // selected row in the entry pane
	offset int // first visible row in the entry pane

	filter textinput.Model

	form formState
	pick pickState

	unsaved int
	message string
	isError bool
}

type formState struct {
	inputs  [4]textinput.Model // ip, hostnames, comment, category
	field   int
	editing *row // nil when adding
	err     string
}

type pickState struct {
	title   string
	options []string
	cursor  int
	apply   func(choice string)
}

type savedMsg struct{ err error }

// Run starts the editor. save writes the file; it is not called in a dry run.
func Run(hf *hosts.HostsFile, cfg *config.Config, dryRun bool, save func(*hosts.HostsFile) error) error {
	_, err := tea.NewProgram(newModel(hf, cfg, dryRun, save), tea.WithAltScreen()).Run()
	return err
}

func newModel(hf *hosts.HostsFile, cfg *config.Config, dryRun bool, save func(*hosts.HostsFile) error) *model {
	filter := textinput.New()
	filter.Prompt = "/"
	filter.Placeholder = "filter by IP, hostname or comment"
	return &model{hf: hf, cfg: cfg, dryRun: dryRun, save: save, filter: filter, focus: focusEntries, width: 100, height: 30}
}

func (m *model) Init() tea.Cmd { return nil }

// ---------------------------------------------------------------- data access

// rows lists the entry pane: matches across every category while filtering,
// otherwise the selected category's entries.
func (m *model) rows() []row {
	var out []row
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	for ci, c := range m.hf.Categories {
		if q == "" && ci != m.cat {
			continue
		}
		for ei, e := range c.Entries {
			if q == "" || matches(e, q) {
				out = append(out, row{ci, ei})
			}
		}
	}
	return out
}

func matches(e hosts.Entry, q string) bool {
	for _, f := range append([]string{e.IP, e.Comment}, e.Hostnames...) {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

func (m *model) entry(r row) *hosts.Entry { return &m.hf.Categories[r.cat].Entries[r.idx] }

func (m *model) selected() (row, bool) {
	rs := m.rows()
	if m.cursor < 0 || m.cursor >= len(rs) {
		return row{}, false
	}
	return rs[m.cursor], true
}

func (m *model) categoryNames() []string {
	names := make([]string, len(m.hf.Categories))
	for i, c := range m.hf.Categories {
		names[i] = c.Name
	}
	return names
}

func enabledCount(c hosts.Category) int {
	n := 0
	for _, e := range c.Entries {
		if e.Enabled {
			n++
		}
	}
	return n
}

// activeProfile names the profile whose categories are exactly the ones with
// enabled entries, or "custom" when none matches.
func (m *model) activeProfile() string {
	var on []string
	for _, c := range m.hf.Categories {
		if enabledCount(c) > 0 {
			on = append(on, c.Name)
		}
	}
	sort.Strings(on)
	for _, name := range m.profileNames() {
		want := slices.Clone(m.cfg.Profiles[name].Categories)
		// A profile may name categories this file does not have.
		want = slices.DeleteFunc(want, func(c string) bool { return m.hf.GetCategory(c) == nil })
		sort.Strings(want)
		if slices.Equal(on, want) {
			return name
		}
	}
	return "custom"
}

func (m *model) profileNames() []string {
	var names []string
	if m.cfg != nil {
		for name := range m.cfg.Profiles {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func (m *model) changed(msg string) {
	m.unsaved++
	m.say(msg)
}

func (m *model) say(msg string)  { m.message, m.isError = msg, false }
func (m *model) fail(msg string) { m.message, m.isError = msg, true }

// ------------------------------------------------------------------- updates

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clamp()
		return m, nil
	case savedMsg:
		if msg.err != nil {
			m.fail("Save failed: " + msg.err.Error())
		} else {
			m.unsaved = 0
			m.say("Saved")
		}
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.mode {
		case modeFilter:
			return m.updateFilter(msg)
		case modeForm:
			return m.updateForm(msg)
		case modePick:
			return m.updatePick(msg)
		case modeHelp:
			m.mode = modeBrowse
			return m, nil
		case modeConfirmQuit:
			return m.updateConfirmQuit(msg)
		}
		return m.updateBrowse(msg)
	}
	return m, nil
}

func (m *model) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	narrow := m.width < narrowWidth
	key := msg.String()
	m.message = "" // a status message lasts until the next key

	// Keys that mean the same in either pane.
	switch key {
	case "q":
		if m.unsaved > 0 {
			m.mode = modeConfirmQuit
			return m, nil
		}
		return m, tea.Quit
	case "s":
		return m, m.saveCmd()
	case "?":
		m.mode = modeHelp
		return m, nil
	case "/":
		m.mode = modeFilter
		m.focus = focusEntries
		return m, m.filter.Focus()
	case "esc":
		m.filter.SetValue("")
		m.cursor, m.offset = 0, 0
		return m, nil
	case "a":
		m.openForm(nil)
		return m, textinput.Blink
	case "p":
		m.openProfilePicker()
		return m, nil
	case "t":
		m.toggleCategory(m.cat)
		return m, nil
	case "tab":
		if !narrow {
			m.focus = 1 - m.focus
		}
		return m, nil
	}

	if m.focus == focusCategories && !narrow {
		switch key {
		case "up", "k":
			m.selectCategory(m.cat - 1)
		case "down", "j":
			m.selectCategory(m.cat + 1)
		case "right", "l", "enter":
			m.focus = focusEntries
		case " ":
			m.toggleCategory(m.cat)
		}
		return m, nil
	}

	switch key {
	case "left", "h":
		if narrow {
			m.selectCategory(m.cat - 1)
		} else {
			m.focus = focusCategories
		}
	case "right", "l":
		if narrow {
			m.selectCategory(m.cat + 1)
		}
	case "up", "k":
		m.cursor--
	case "down", "j":
		m.cursor++
	case "pgup":
		m.cursor -= m.listHeight()
	case "pgdown":
		m.cursor += m.listHeight()
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = len(m.rows()) - 1
	case " ":
		if r, ok := m.selected(); ok {
			e := m.entry(r)
			e.Enabled = !e.Enabled
			state := "Disabled"
			if e.Enabled {
				state = "Enabled"
			}
			m.changed(fmt.Sprintf("%s %s", state, strings.Join(e.Hostnames, " ")))
		}
	case "d":
		if r, ok := m.selected(); ok {
			c := &m.hf.Categories[r.cat]
			name := strings.Join(c.Entries[r.idx].Hostnames, " ")
			c.Entries = slices.Delete(c.Entries, r.idx, r.idx+1)
			m.changed("Deleted " + name)
		}
	case "e", "enter":
		if r, ok := m.selected(); ok {
			m.openForm(&r)
			return m, textinput.Blink
		}
	case "m":
		if r, ok := m.selected(); ok {
			m.openMovePicker(r)
		}
	}
	m.clamp()
	return m, nil
}

func (m *model) selectCategory(i int) {
	if n := len(m.hf.Categories); n > 0 {
		m.cat = (i + n) % n
	}
	m.cursor, m.offset = 0, 0
}

// toggleCategory disables every entry in the category if any is enabled,
// otherwise enables them all.
func (m *model) toggleCategory(i int) {
	if i < 0 || i >= len(m.hf.Categories) {
		return
	}
	c := m.hf.Categories[i]
	if enabledCount(c) > 0 {
		m.hf.DisableCategory(c.Name)
		m.changed("Disabled category " + c.Name)
	} else {
		m.hf.EnableCategory(c.Name)
		m.changed("Enabled category " + c.Name)
	}
}

func (m *model) saveCmd() tea.Cmd {
	if m.dryRun {
		m.say("Dry run: changes not saved")
		return nil
	}
	hf, save := m.hf, m.save
	return func() tea.Msg { return savedMsg{save(hf)} }
}

// clamp keeps the cursor on a row and the row on screen.
func (m *model) clamp() {
	n := len(m.rows())
	m.cursor = max(0, min(m.cursor, n-1))
	h := m.listHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	m.offset = max(0, min(m.offset, max(0, n-h)))
}

func (m *model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.mode = modeBrowse
		m.filter.Blur()
		return m, nil
	case "esc":
		m.mode = modeBrowse
		m.filter.Blur()
		m.filter.SetValue("")
		m.cursor, m.offset = 0, 0
		return m, nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.cursor, m.offset = 0, 0
	return m, cmd
}

func (m *model) updateConfirmQuit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s":
		if m.dryRun {
			return m, tea.Quit
		}
		if err := m.save(m.hf); err != nil {
			m.mode = modeBrowse
			m.fail("Save failed: " + err.Error())
			return m, nil
		}
		return m, tea.Quit
	case "q":
		return m, tea.Quit
	case "esc":
		m.mode = modeBrowse
	}
	return m, nil
}

// --------------------------------------------------------------------- forms

var formLabels = [4]string{"IP", "Hostnames", "Comment", "Category"}

func (m *model) openForm(editing *row) {
	var f formState
	for i := range f.inputs {
		f.inputs[i] = textinput.New()
		f.inputs[i].Prompt = ""
		f.inputs[i].CharLimit = 500
	}
	f.inputs[1].Placeholder = "one or more, separated by spaces"
	f.inputs[2].Placeholder = "optional"

	switch {
	case editing != nil:
		e := m.entry(*editing)
		f.editing = editing
		f.inputs[0].SetValue(e.IP)
		f.inputs[1].SetValue(strings.Join(e.Hostnames, " "))
		f.inputs[2].SetValue(e.Comment)
		f.inputs[3].SetValue(m.hf.Categories[editing.cat].Name)
	case len(m.hf.Categories) > 0:
		f.inputs[3].SetValue(m.hf.Categories[m.cat].Name)
	case m.cfg != nil:
		f.inputs[3].SetValue(m.cfg.General.DefaultCategory)
	}
	f.inputs[0].Focus()
	m.form = f
	m.mode = modeForm
}

// formEntry builds the entry the form describes. For an edit it starts from
// the original so that fields the form does not show (notes, enabled) survive.
func (m *model) formEntry() hosts.Entry {
	var e hosts.Entry
	if m.form.editing != nil {
		e = *m.entry(*m.form.editing)
	} else {
		e.Enabled = true
	}
	e.IP = strings.TrimSpace(m.form.inputs[0].Value())
	e.Hostnames = strings.Fields(m.form.inputs[1].Value())
	e.Comment = strings.TrimSpace(m.form.inputs[2].Value())
	e.Category = strings.TrimSpace(m.form.inputs[3].Value())
	return e
}

func (m *model) validateForm() {
	e := m.formEntry()
	if e.IP == "" || len(e.Hostnames) == 0 {
		m.form.err = ""
		return
	}
	if err := hosts.ValidateEntry(e); err != nil {
		m.form.err = err.Error()
	} else {
		m.form.err = ""
	}
}

func (m *model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	f := &m.form
	switch msg.String() {
	case "esc":
		m.mode = modeBrowse
		return m, nil
	case "tab", "down":
		return m, m.focusField(f.field + 1)
	case "shift+tab", "up":
		return m, m.focusField(f.field + 3)
	case "enter":
		m.submitForm()
		return m, nil
	}
	var cmd tea.Cmd
	f.inputs[f.field], cmd = f.inputs[f.field].Update(msg)
	m.validateForm()
	return m, cmd
}

func (m *model) focusField(i int) tea.Cmd {
	m.form.inputs[m.form.field].Blur()
	m.form.field = i % len(m.form.inputs)
	return m.form.inputs[m.form.field].Focus()
}

func (m *model) submitForm() {
	e := m.formEntry()
	if e.IP == "" || len(e.Hostnames) == 0 {
		m.form.err = "IP and at least one hostname are required"
		return
	}
	if err := hosts.ValidateEntry(e); err != nil {
		m.form.err = err.Error()
		return
	}

	if r := m.form.editing; r != nil && m.hf.Categories[r.cat].Name == e.Category {
		*m.entry(*r) = e
		m.changed("Updated " + strings.Join(e.Hostnames, " "))
	} else {
		if r != nil {
			c := &m.hf.Categories[r.cat]
			c.Entries = slices.Delete(c.Entries, r.idx, r.idx+1)
		}
		if err := m.hf.AddEntry(e); err != nil {
			m.form.err = err.Error()
			return
		}
		verb := "Added"
		if r != nil {
			verb = "Moved"
		}
		m.changed(fmt.Sprintf("%s %s in %s", verb, strings.Join(e.Hostnames, " "), e.Category))
		m.showEntry(e.Category, -1)
	}
	m.mode = modeBrowse
	m.clamp()
}

// showEntry selects category name and, with idx < 0, its last entry.
func (m *model) showEntry(name string, idx int) {
	m.filter.SetValue("")
	for i, c := range m.hf.Categories {
		if c.Name == name {
			m.cat = i
			if idx < 0 {
				idx = len(c.Entries) - 1
			}
			m.cursor = idx
			m.focus = focusEntries
			return
		}
	}
}

// ------------------------------------------------------------------- pickers

func (m *model) openMovePicker(r row) {
	from := m.hf.Categories[r.cat].Name
	var options []string
	for _, name := range m.categoryNames() {
		if name != from {
			options = append(options, name)
		}
	}
	if len(options) == 0 {
		m.fail("No other category to move to; edit the entry to type a new one")
		return
	}
	m.pick = pickState{title: "Move to category", options: options, apply: func(to string) {
		e := *m.entry(r)
		e.Category = to
		c := &m.hf.Categories[r.cat]
		c.Entries = slices.Delete(c.Entries, r.idx, r.idx+1)
		target := m.hf.GetCategory(to)
		target.Entries = append(target.Entries, e)
		m.changed(fmt.Sprintf("Moved %s to %s", strings.Join(e.Hostnames, " "), to))
		m.showEntry(to, -1)
	}}
	m.mode = modePick
}

func (m *model) openProfilePicker() {
	names := m.profileNames()
	if len(names) == 0 {
		m.fail("No profiles defined in the config file")
		return
	}
	m.pick = pickState{title: "Activate profile", options: names, apply: func(name string) {
		m.hf.SetEnabledCategories(m.cfg.Profiles[name].Categories)
		m.changed("Activated profile " + name)
	}}
	if i := slices.Index(names, m.activeProfile()); i >= 0 {
		m.pick.cursor = i
	}
	m.mode = modePick
}

func (m *model) updatePick(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := &m.pick
	switch msg.String() {
	case "esc", "q":
		m.mode = modeBrowse
	case "up", "k":
		p.cursor = (p.cursor + len(p.options) - 1) % len(p.options)
	case "down", "j":
		p.cursor = (p.cursor + 1) % len(p.options)
	case "enter", " ":
		m.mode = modeBrowse
		p.apply(p.options[p.cursor])
		m.clamp()
	}
	return m, nil
}

// --------------------------------------------------------------------- views

var (
	accent    = lipgloss.Color("39")
	dim       = lipgloss.Color("243")
	green     = lipgloss.Color("76")
	red       = lipgloss.Color("203")
	yellow    = lipgloss.Color("214")
	titleSt   = lipgloss.NewStyle().Bold(true).Foreground(accent)
	dimSt     = lipgloss.NewStyle().Foreground(dim)
	onSt      = lipgloss.NewStyle().Foreground(green)
	errSt     = lipgloss.NewStyle().Foreground(red)
	warnSt    = lipgloss.NewStyle().Foreground(yellow).Bold(true)
	cursorSt  = lipgloss.NewStyle().Reverse(true)
	keySt     = lipgloss.NewStyle().Bold(true)
	paneSt    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(dim)
	activeSt  = paneSt.BorderForeground(accent)
	paneTitle = lipgloss.NewStyle().Bold(true)
)

// listHeight is the number of entry rows that fit in the entry pane.
func (m *model) listHeight() int {
	// header + footer + two border lines + pane title + scroll line
	return max(1, m.height-6)
}

func (m *model) View() string {
	header := m.viewHeader()
	footer := m.viewFooter()
	bodyHeight := max(3, m.height-2)

	var body string
	if m.width < narrowWidth {
		body = m.viewMain(m.width, bodyHeight, true)
	} else {
		left := m.viewCategories(sidebarWidth, bodyHeight)
		right := m.viewMain(m.width-sidebarWidth, bodyHeight, false)
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}
	// The bars must not wrap: a wrapped line pushes the frame off screen.
	header = ansi.Truncate(header, m.width, "…")
	footer = ansi.Truncate(footer, m.width, "…")
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m *model) viewHeader() string {
	left := titleSt.Render(" hosts-manager ") + dimSt.Render(m.hf.FilePath)
	right := "profile: " + m.activeProfile()
	if m.dryRun {
		right = warnSt.Render("dry run") + "  " + right
	}
	if m.unsaved > 0 {
		right += "  " + warnSt.Render(fmt.Sprintf("● %d unsaved", m.unsaved))
	}
	gap := max(1, m.width-lipgloss.Width(left)-lipgloss.Width(right)-1)
	return left + strings.Repeat(" ", gap) + right
}

func (m *model) viewFooter() string {
	if m.mode == modeConfirmQuit {
		return warnSt.Render(fmt.Sprintf(" %d unsaved change(s) — ", m.unsaved)) +
			keys("s", "save & quit", "q", "discard & quit", "esc", "stay")
	}
	if m.message != "" {
		if m.isError {
			return " " + errSt.Render(m.message)
		}
		return " " + onSt.Render(m.message)
	}
	if m.mode == modeForm {
		return keys("tab", "next field", "enter", "save", "esc", "cancel")
	}
	if m.mode == modePick {
		return keys("↑↓", "choose", "enter", "apply", "esc", "cancel")
	}
	if m.width < narrowWidth {
		return keys("space", "toggle", "←→", "category", "a", "add", "e", "edit", "s", "save", "?", "help")
	}
	if m.focus == focusCategories {
		return keys("space", "toggle category", "→", "entries", "a", "add", "p", "profile", "s", "save", "?", "help", "q", "quit")
	}
	return keys("space", "toggle", "a", "add", "e", "edit", "d", "delete", "m", "move", "/", "filter", "s", "save", "?", "help")
}

func keys(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, keySt.Render(pairs[i])+" "+dimSt.Render(pairs[i+1]))
	}
	return " " + strings.Join(parts, dimSt.Render(" · "))
}

func (m *model) viewCategories(width, height int) string {
	var lines []string
	for i, c := range m.hf.Categories {
		on := enabledCount(c)
		mark := dimSt.Render("○")
		if on > 0 {
			mark = onSt.Render("●")
		}
		count := fmt.Sprintf("%d/%d", on, len(c.Entries))
		name := truncate(c.Name, width-len(count)-7)
		line := fmt.Sprintf(" %s %s%s%s ", mark, name,
			strings.Repeat(" ", max(1, width-6-lipgloss.Width(name)-len(count))), count)
		if i == m.cat && m.filter.Value() == "" {
			if m.focus == focusCategories {
				line = cursorSt.Render(line)
			} else {
				line = paneTitle.Render(line)
			}
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, dimSt.Render(" none yet — press a"))
	}
	lines = lines[:min(len(lines), max(1, height-2))]
	st := paneSt
	if m.focus == focusCategories {
		st = activeSt
	}
	return st.Width(width - 2).Height(height - 2).Render(strings.Join(lines, "\n"))
}

func (m *model) viewMain(width, height int, narrow bool) string {
	inner := width - 2
	var content string
	switch m.mode {
	case modeForm:
		content = m.viewForm(inner)
	case modePick:
		content = m.viewPick()
	case modeHelp:
		content = viewHelp()
	default:
		content = m.viewEntries(inner, narrow)
	}
	st := paneSt
	if m.focus == focusEntries || narrow || m.mode != modeBrowse {
		st = activeSt
	}
	return st.Width(inner).Height(height - 2).Render(content)
}

func (m *model) viewEntries(width int, narrow bool) string {
	var title string
	switch {
	case m.mode == modeFilter || m.filter.Value() != "":
		title = m.filter.View()
	case len(m.hf.Categories) == 0:
		title = "no entries"
	case narrow:
		title = fmt.Sprintf("‹ %s ›", m.hf.Categories[m.cat].Name)
	default:
		title = m.hf.Categories[m.cat].Name
	}
	lines := []string{paneTitle.Render(" " + title)}

	rs := m.rows()
	filtering := m.filter.Value() != ""
	ipWidth := 15
	for _, r := range rs {
		ipWidth = max(ipWidth, len(m.entry(r).IP))
	}

	end := min(len(rs), m.offset+m.listHeight())
	for i := m.offset; i < end; i++ {
		e := m.entry(rs[i])
		mark, st := onSt.Render("●"), lipgloss.NewStyle()
		if !e.Enabled {
			mark, st = dimSt.Render("○"), dimSt
		}
		text := fmt.Sprintf("%-*s  %s", ipWidth, e.IP, strings.Join(e.Hostnames, " "))
		if filtering {
			text = fmt.Sprintf("%-12s ", truncate(m.hf.Categories[rs[i].cat].Name, 12)) + text
		}
		if e.Comment != "" {
			text += "  # " + e.Comment
		}
		text = truncate(text, width-5)
		pointer := "  "
		if i == m.cursor && m.focus == focusEntries {
			pointer = "▸ "
			text = cursorSt.Render(text)
		} else {
			text = st.Render(text)
		}
		lines = append(lines, pointer+mark+" "+text)
	}
	if len(rs) == 0 {
		lines = append(lines, dimSt.Render("  nothing here — press a to add an entry"))
	}
	if len(rs) > m.listHeight() {
		lines = append(lines, dimSt.Render(fmt.Sprintf("%*s", width-1,
			fmt.Sprintf("%d–%d of %d", m.offset+1, end, len(rs)))))
	}
	return strings.Join(lines, "\n")
}

func (m *model) viewForm(width int) string {
	title := "Add entry"
	if m.form.editing != nil {
		title = "Edit entry"
	}
	lines := []string{paneTitle.Render(" " + title), ""}
	for i, in := range m.form.inputs {
		label := fmt.Sprintf(" %-10s ", formLabels[i])
		if i == m.form.field {
			label = keySt.Render(label)
		} else {
			label = dimSt.Render(label)
		}
		in.Width = max(10, width-14)
		lines = append(lines, label+in.View())
	}
	if m.form.field == 3 && len(m.hf.Categories) > 0 {
		lines = append(lines, dimSt.Render(truncate("            existing: "+strings.Join(m.categoryNames(), ", "), width)))
	}
	lines = append(lines, "")
	if m.form.err != "" {
		lines = append(lines, errSt.Render(" ✗ "+truncate(m.form.err, width-4)))
	}
	return strings.Join(lines, "\n")
}

func (m *model) viewPick() string {
	lines := []string{paneTitle.Render(" " + m.pick.title), ""}
	for i, o := range m.pick.options {
		if i == m.pick.cursor {
			lines = append(lines, "▸ "+cursorSt.Render(o))
		} else {
			lines = append(lines, "  "+o)
		}
	}
	return strings.Join(lines, "\n")
}

func viewHelp() string {
	rows := [][2]string{
		{"tab / ← →", "switch pane (narrow terminal: switch category)"},
		{"↑ ↓  k j", "move"},
		{"pgup pgdn  g G", "page, top, bottom"},
		{"space", "enable/disable entry, or category in the left pane"},
		{"t", "enable/disable the whole category"},
		{"a  e  d", "add, edit, delete entry"},
		{"m", "move entry to another category"},
		{"/  esc", "filter all categories, clear filter"},
		{"p", "activate a profile"},
		{"s", "save (backs up the current file first)"},
		{"q", "quit (asks if there are unsaved changes)"},
	}
	lines := []string{paneTitle.Render(" Keys"), ""}
	for _, r := range rows {
		lines = append(lines, fmt.Sprintf(" %s %s", keySt.Render(fmt.Sprintf("%-15s", r[0])), r[1]))
	}
	lines = append(lines, "", dimSt.Render(" Changes stay in memory until you save. Any key closes this."))
	return strings.Join(lines, "\n")
}

func truncate(s string, n int) string {
	if n <= 1 || lipgloss.Width(s) <= n {
		return s
	}
	r := []rune(s)
	if len(r) > n-1 {
		r = r[:n-1]
	}
	return string(r) + "…"
}
