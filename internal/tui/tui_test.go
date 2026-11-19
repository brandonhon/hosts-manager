package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/brandonhon/hosts-manager/internal/config"
	"github.com/brandonhon/hosts-manager/internal/hosts"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func testFile() *hosts.HostsFile {
	return &hosts.HostsFile{
		FilePath: "/etc/hosts",
		Categories: []hosts.Category{
			{Name: "dev", Enabled: true, Entries: []hosts.Entry{
				{IP: "10.0.0.1", Hostnames: []string{"api.dev"}, Category: "dev", Enabled: true},
				{IP: "10.0.0.2", Hostnames: []string{"web.dev"}, Category: "dev", Enabled: false, Notes: []string{"# keep"}},
			}},
			{Name: "prod", Enabled: true, Entries: []hosts.Entry{
				{IP: "203.0.113.1", Hostnames: []string{"api.example.com"}, Category: "prod", Enabled: true},
			}},
		},
	}
}

func testModel(hf *hosts.HostsFile) *model {
	cfg := config.DefaultConfig()
	cfg.Profiles = map[string]config.Profile{
		"devonly": {Categories: []string{"dev"}, Default: true},
		"all":     {Categories: []string{"dev", "prod"}},
	}
	m := newModel(hf, cfg, false, func(*hosts.HostsFile) error { return nil })
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	return m
}

// press sends keys: named ones ("esc", "enter", "tab", "space", "ctrl+u", ...)
// or literal text, one rune at a time.
func press(m *model, keys ...string) tea.Cmd {
	named := map[string]tea.KeyType{
		"esc": tea.KeyEsc, "enter": tea.KeyEnter, "tab": tea.KeyTab, "space": tea.KeySpace,
		"ctrl+c": tea.KeyCtrlC, "pgdown": tea.KeyPgDown, "backspace": tea.KeyBackspace,
		"ctrl+u": tea.KeyCtrlU,
	}
	var cmd tea.Cmd
	for _, k := range keys {
		if t, ok := named[k]; ok {
			_, cmd = m.Update(tea.KeyMsg{Type: t})
			continue
		}
		for _, r := range k {
			_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
	}
	return cmd
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// The old editor stored the window size but never used it, so a long list ran
// off the screen and the cursor with it.
func TestLongListScrollsWithCursor(t *testing.T) {
	hf := &hosts.HostsFile{Categories: []hosts.Category{{Name: "many", Enabled: true}}}
	for i := 0; i < 60; i++ {
		hf.Categories[0].Entries = append(hf.Categories[0].Entries,
			hosts.Entry{IP: "10.0.0.1", Hostnames: []string{fmt.Sprintf("host%02d.test", i)}, Enabled: true})
	}
	m := testModel(hf)

	for i := 0; i < 45; i++ {
		press(m, "j")
	}
	if m.cursor != 45 || m.cursor < m.offset || m.cursor >= m.offset+m.listHeight() {
		t.Fatalf("cursor %d not visible in rows %d..%d", m.cursor, m.offset, m.offset+m.listHeight())
	}
	view := m.View()
	if !strings.Contains(view, "host45.test") || strings.Contains(view, "host00.test") {
		t.Error("view does not follow the cursor")
	}
	if got := strings.Count(view, "\n") + 1; got > 20 {
		t.Errorf("view is %d lines, taller than the 20-line window", got)
	}
}

// The old editor's q quit immediately, discarding unsaved edits.
func TestQuitAsksWhenThereAreUnsavedChanges(t *testing.T) {
	m := testModel(testFile())
	press(m, "space")
	if cmd := press(m, "q"); isQuit(cmd) || m.mode != modeConfirmQuit {
		t.Fatal("q with unsaved changes should ask, not quit")
	}
	press(m, "esc")
	if m.mode != modeBrowse || m.unsaved != 1 {
		t.Fatal("esc should return to the editor with the change kept")
	}
	press(m, "q")
	if !isQuit(press(m, "q")) {
		t.Fatal("q, q should discard and quit")
	}

	clean := testModel(testFile())
	if !isQuit(press(clean, "q")) {
		t.Fatal("q with nothing unsaved should quit at once")
	}
}

func TestSaveAndQuitSaves(t *testing.T) {
	m := testModel(testFile())
	saved := 0
	m.save = func(*hosts.HostsFile) error { saved++; return nil }
	press(m, "space", "q")
	if !isQuit(press(m, "s")) || saved != 1 {
		t.Fatalf("s at the quit prompt should save once and quit, saved=%d", saved)
	}
}

// The old editor found the entry to toggle, edit or delete by IP and first
// hostname, so of two such lines it always acted on the first.
func TestDuplicateLinesAreDistinctRows(t *testing.T) {
	dup := func() *hosts.HostsFile {
		return &hosts.HostsFile{Categories: []hosts.Category{{Name: "lb", Enabled: true, Entries: []hosts.Entry{
			{IP: "10.125.12.85", Hostnames: []string{"www.example.com"}, Comment: "lb01", Enabled: true},
			{IP: "10.125.12.85", Hostnames: []string{"www.example.com"}, Comment: "lb02", Enabled: true},
		}}}}
	}

	m := testModel(dup())
	press(m, "j", "space")
	es := m.hf.Categories[0].Entries
	if !es[0].Enabled || es[1].Enabled {
		t.Errorf("toggling the second line changed the wrong one: %+v", es)
	}

	m = testModel(dup())
	press(m, "j", "d")
	es = m.hf.Categories[0].Entries
	if len(es) != 1 || es[0].Comment != "lb01" {
		t.Errorf("deleting the second line removed the wrong one: %+v", es)
	}
}

// The old editor wrote edited values without validating them.
func TestEditRejectsInvalidValues(t *testing.T) {
	m := testModel(testFile())
	press(m, "e", "tab", "ctrl+u", "bad_host!", "enter")
	if m.mode != modeForm || !strings.Contains(m.form.err, "hostname") {
		t.Fatalf("expected a hostname error in the form, got mode=%v err=%q", m.mode, m.form.err)
	}
	if got := m.hf.Categories[0].Entries[0].Hostnames[0]; got != "api.dev" {
		t.Errorf("invalid edit was applied: %s", got)
	}
}

func TestFormShowsErrorsAsYouType(t *testing.T) {
	m := testModel(testFile())
	press(m, "a", "999.1.1.1", "tab", "x.test")
	if !strings.Contains(m.form.err, "IP") {
		t.Errorf("expected a live IP error, got %q", m.form.err)
	}
	press(m, "tab", "tab", "tab") // back to IP
	press(m, "ctrl+u", "10.1.1.1")
	if m.form.err != "" {
		t.Errorf("error should clear once valid, got %q", m.form.err)
	}
}

func TestEditKeepsNotesAndState(t *testing.T) {
	m := testModel(testFile())
	press(m, "j", "e", "tab", "tab", "new comment", "enter")
	e := m.hf.Categories[0].Entries[1]
	if e.Comment != "new comment" || e.Enabled || len(e.Notes) != 1 {
		t.Errorf("edit lost fields: %+v", e)
	}
}

func TestAddToNewCategoryCreatesAndSelectsIt(t *testing.T) {
	m := testModel(testFile())
	press(m, "a", "10.9.9.9", "tab", "lab.test", "tab", "tab", "ctrl+u", "lab", "enter")
	if m.mode != modeBrowse {
		t.Fatalf("form did not close: %q", m.form.err)
	}
	c := m.hf.GetCategory("lab")
	if c == nil || len(c.Entries) != 1 || m.hf.Categories[m.cat].Name != "lab" {
		t.Fatalf("new category not created and selected: %+v", m.hf.Categories)
	}
	if m.unsaved != 1 {
		t.Errorf("unsaved = %d, want 1", m.unsaved)
	}
}

func TestEditingCategoryMovesEntry(t *testing.T) {
	m := testModel(testFile())
	press(m, "e", "tab", "tab", "tab", "ctrl+u", "prod", "enter")
	if len(m.hf.GetCategory("dev").Entries) != 1 || len(m.hf.GetCategory("prod").Entries) != 2 {
		t.Errorf("entry not moved: %+v", m.hf.Categories)
	}
}

func TestMovePicker(t *testing.T) {
	m := testModel(testFile())
	press(m, "m")
	if m.mode != modePick || len(m.pick.options) != 1 || m.pick.options[0] != "prod" {
		t.Fatalf("picker should offer the other categories: %+v", m.pick.options)
	}
	press(m, "enter")
	if len(m.hf.GetCategory("prod").Entries) != 2 || m.hf.Categories[m.cat].Name != "prod" {
		t.Errorf("move not applied or not followed: %+v", m.hf.Categories)
	}
}

func TestCategoryPaneToggle(t *testing.T) {
	m := testModel(testFile())
	press(m, "tab", "space") // dev has one enabled entry -> disable all
	if enabledCount(m.hf.Categories[0]) != 0 {
		t.Fatal("space on a category with enabled entries should disable them all")
	}
	press(m, "space")
	if enabledCount(m.hf.Categories[0]) != 2 {
		t.Fatal("space on a fully disabled category should enable it")
	}
	press(m, "j")
	if m.cat != 1 {
		t.Errorf("j in the category pane should select the next category")
	}
}

func TestFilterSearchesEveryCategory(t *testing.T) {
	m := testModel(testFile())
	press(m, "/", "api")
	if rs := m.rows(); len(rs) != 2 {
		t.Fatalf("filter should match api.dev and api.example.com, got %d rows", len(rs))
	}
	press(m, "enter", "j", "space")
	if m.hf.Categories[1].Entries[0].Enabled {
		t.Error("toggling a filtered row should change that entry")
	}
	press(m, "esc")
	if m.filter.Value() != "" || len(m.rows()) != 2 {
		t.Error("esc should clear the filter and show the selected category")
	}
}

func TestProfilePickerAndActiveProfile(t *testing.T) {
	m := testModel(testFile())
	if got := m.activeProfile(); got != "all" {
		t.Fatalf("active profile = %q, want all", got)
	}
	press(m, "p")
	for m.pick.options[m.pick.cursor] != "devonly" {
		press(m, "j")
	}
	press(m, "enter")
	if enabledCount(m.hf.Categories[1]) != 0 || enabledCount(m.hf.Categories[0]) != 2 {
		t.Errorf("profile not applied: %+v", m.hf.Categories)
	}
	if got := m.activeProfile(); got != "devonly" {
		t.Errorf("active profile = %q, want devonly", got)
	}
}

func TestSaveUsesSaveFuncAndReportsErrors(t *testing.T) {
	m := testModel(testFile())
	var got *hosts.HostsFile
	m.save = func(hf *hosts.HostsFile) error { got = hf; return nil }
	press(m, "space")
	m.Update(press(m, "s")())
	if got != m.hf || m.unsaved != 0 {
		t.Fatal("save func not called with the file, or unsaved not reset")
	}

	m.save = func(*hosts.HostsFile) error { return errors.New("permission denied") }
	press(m, "space")
	m.Update(press(m, "s")())
	if m.unsaved != 1 || !m.isError || !strings.Contains(m.message, "permission denied") {
		t.Errorf("failed save should keep changes and show the error: %q", m.message)
	}
}

// A dry run skips the privilege check, so saving must not write.
func TestDryRunSaveDoesNotWrite(t *testing.T) {
	m := testModel(testFile())
	m.dryRun = true
	m.save = func(*hosts.HostsFile) error { t.Fatal("save called in a dry run"); return nil }
	press(m, "space")
	if cmd := press(m, "s"); cmd != nil {
		t.Fatal("dry-run save should not schedule a write")
	}
	if !strings.Contains(m.message, "Dry run") {
		t.Errorf("message = %q", m.message)
	}
	press(m, "q")
	if !isQuit(press(m, "s")) {
		t.Error("s at the quit prompt in a dry run should quit without saving")
	}
}

func TestNarrowTerminalHidesCategoryPane(t *testing.T) {
	m := testModel(testFile())
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	if v := m.View(); !strings.Contains(v, "‹ dev ›") || strings.Contains(v, "1/2") {
		t.Errorf("narrow view should show the category in the title and no category pane:\n%s", v)
	}
	for i, line := range strings.Split(m.View(), "\n") {
		if w := lipgloss.Width(line); w > 60 {
			t.Errorf("line %d is %d wide in a 60-column terminal: %q", i, w, line)
		}
	}
	press(m, "l")
	if m.hf.Categories[m.cat].Name != "prod" {
		t.Error("→ in a narrow terminal should switch category")
	}
}

func TestWideViewShowsBothPanes(t *testing.T) {
	m := testModel(testFile())
	v := m.View()
	for _, want := range []string{"hosts-manager", "/etc/hosts", "profile: all", "dev", "1/2", "prod", "api.dev", "space toggle"} {
		if !strings.Contains(v, want) {
			t.Errorf("view missing %q", want)
		}
	}
	press(m, "space")
	if !strings.Contains(m.View(), "1 unsaved") {
		t.Error("header should count unsaved changes")
	}
}
