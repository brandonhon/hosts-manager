package tui

import (
	"strings"
	"testing"

	"hosts-manager/internal/config"
	"hosts-manager/internal/hosts"

	tea "github.com/charmbracelet/bubbletea"
)

// createTestModel creates a test model with sample data
func createTestModel() *model {
	// Create test config
	cfg := &config.Config{
		General: config.General{
			DefaultCategory: "development",
		},
	}

	// Create test hosts file with multiple categories
	hostsFile := &hosts.HostsFile{
		Categories: []hosts.Category{
			{
				Name:        "development",
				Description: "Development hosts",
				Enabled:     true,
				Entries: []hosts.Entry{
					{
						IP:        "127.0.0.1",
						Hostnames: []string{"dev.local"},
						Comment:   "Dev server",
						Category:  "development",
						Enabled:   true,
					},
					{
						IP:        "192.168.1.100",
						Hostnames: []string{"api.dev"},
						Comment:   "API server",
						Category:  "development",
						Enabled:   true,
					},
				},
			},
			{
				Name:        "staging",
				Description: "Staging hosts",
				Enabled:     true,
				Entries: []hosts.Entry{
					{
						IP:        "10.0.1.50",
						Hostnames: []string{"staging.local"},
						Comment:   "Staging server",
						Category:  "staging",
						Enabled:   true,
					},
				},
			},
			{
				Name:        "production",
				Description: "Production hosts",
				Enabled:     true,
				Entries: []hosts.Entry{
					{
						IP:        "203.0.113.10",
						Hostnames: []string{"prod.example.com"},
						Comment:   "Production server",
						Category:  "production",
						Enabled:   true,
					},
				},
			},
		},
		FilePath: "/tmp/test-hosts",
	}

	m := &model{
		hostsFile:   hostsFile,
		config:      cfg,
		currentView: viewMain,
		selected:    make(map[int]bool),
		entries:     buildEntryList(hostsFile),
		categories:  []string{"development", "staging", "production"},
	}

	return m
}

func TestBuildEntryList(t *testing.T) {
	m := createTestModel()

	if len(m.entries) != 4 {
		t.Errorf("Expected 4 entries, got %d", len(m.entries))
	}

	// Check first entry
	if m.entries[0].entry.Hostnames[0] != "dev.local" {
		t.Errorf("Expected first entry hostname to be 'dev.local', got '%s'", m.entries[0].entry.Hostnames[0])
	}
	if m.entries[0].category != "development" {
		t.Errorf("Expected first entry category to be 'development', got '%s'", m.entries[0].category)
	}

	// Check last entry
	lastIndex := len(m.entries) - 1
	if m.entries[lastIndex].entry.Hostnames[0] != "prod.example.com" {
		t.Errorf("Expected last entry hostname to be 'prod.example.com', got '%s'", m.entries[lastIndex].entry.Hostnames[0])
	}
	if m.entries[lastIndex].category != "production" {
		t.Errorf("Expected last entry category to be 'production', got '%s'", m.entries[lastIndex].category)
	}
}

func TestGetAvailableCategories(t *testing.T) {
	m := createTestModel()

	// Test with first entry (development category)
	m.moveEntryIndex = 0
	available := m.getAvailableCategories()

	if len(available) != 2 {
		t.Errorf("Expected 2 available categories, got %d", len(available))
	}

	// Should not include current category (development)
	for _, cat := range available {
		if cat == "development" {
			t.Errorf("Available categories should not include current category 'development'")
		}
	}

	// Should include staging and production
	expectedCategories := map[string]bool{"staging": false, "production": false}
	for _, cat := range available {
		if _, exists := expectedCategories[cat]; exists {
			expectedCategories[cat] = true
		}
	}
	for cat, found := range expectedCategories {
		if !found {
			t.Errorf("Expected category '%s' to be available", cat)
		}
	}
}

func TestMoveEntry(t *testing.T) {
	m := createTestModel()

	// Test moving first entry from development to staging
	entryIndex := 0
	targetCategory := "staging"

	// Get original counts
	devCountBefore := len(m.hostsFile.Categories[0].Entries)
	stagingCountBefore := len(m.hostsFile.Categories[1].Entries)

	// Perform move
	err := m.moveEntry(entryIndex, targetCategory)
	if err != nil {
		t.Errorf("Move operation failed: %v", err)
	}

	// Check counts after move
	devCountAfter := len(m.hostsFile.Categories[0].Entries)
	stagingCountAfter := len(m.hostsFile.Categories[1].Entries)

	if devCountAfter != devCountBefore-1 {
		t.Errorf("Expected development category to have %d entries, got %d", devCountBefore-1, devCountAfter)
	}
	if stagingCountAfter != stagingCountBefore+1 {
		t.Errorf("Expected staging category to have %d entries, got %d", stagingCountBefore+1, stagingCountAfter)
	}

	// Check that the moved entry has the correct category
	found := false
	for _, entry := range m.hostsFile.Categories[1].Entries {
		if entry.Hostnames[0] == "dev.local" && entry.Category == "staging" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Moved entry not found in target category with correct category assignment")
	}

	// Check that entry was removed from source category
	for _, entry := range m.hostsFile.Categories[0].Entries {
		if entry.Hostnames[0] == "dev.local" {
			t.Errorf("Entry should have been removed from source category")
		}
	}
}

func TestMoveEntryErrors(t *testing.T) {
	m := createTestModel()

	// Test invalid entry index
	err := m.moveEntry(999, "staging")
	if err == nil {
		t.Errorf("Expected error for invalid entry index")
	}

	// Test invalid target category
	err = m.moveEntry(0, "nonexistent")
	if err == nil {
		t.Errorf("Expected error for invalid target category")
	}
}

func TestFindEntryAfterMove(t *testing.T) {
	m := createTestModel()

	// Move an entry first
	entryIndex := 0
	targetCategory := "staging"
	entryToMove := m.entries[entryIndex]

	err := m.moveEntry(entryIndex, targetCategory)
	if err != nil {
		t.Fatalf("Failed to move entry for test: %v", err)
	}

	// Rebuild entry list
	m.entries = buildEntryList(m.hostsFile)

	// Find the moved entry
	newIndex := m.findEntryAfterMove(entryToMove, targetCategory)

	if newIndex < 0 || newIndex >= len(m.entries) {
		t.Errorf("findEntryAfterMove returned invalid index: %d", newIndex)
	}

	// Verify the found entry is correct
	foundEntry := m.entries[newIndex]
	if foundEntry.entry.Hostnames[0] != "dev.local" || foundEntry.category != "staging" {
		t.Errorf("findEntryAfterMove found wrong entry: %+v", foundEntry)
	}
}

func TestUpdateMoveNavigation(t *testing.T) {
	m := createTestModel()

	// Set up move mode
	m.currentView = viewMove
	m.moveEntryIndex = 0
	m.moveCategoryCursor = 0
	m.moveTargetCategory = "staging"

	// Test down navigation
	newModel, _ := m.updateMove(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(*model)

	if m.moveCategoryCursor != 1 {
		t.Errorf("Expected cursor to move to 1, got %d", m.moveCategoryCursor)
	}

	// Test up navigation
	newModel, _ = m.updateMove(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(*model)

	if m.moveCategoryCursor != 0 {
		t.Errorf("Expected cursor to move back to 0, got %d", m.moveCategoryCursor)
	}

	// Test escape (cancel move)
	newModel, _ = m.updateMove(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(*model)

	if m.currentView != viewMain {
		t.Errorf("Expected view to return to main after escape")
	}
}

func TestUpdateMoveExecution(t *testing.T) {
	m := createTestModel()

	// Set up move mode
	m.currentView = viewMove
	m.moveEntryIndex = 0 // First entry (dev.local in development)
	m.moveCategoryCursor = 0
	m.moveTargetCategory = "staging"

	// Get original counts
	devCountBefore := len(m.hostsFile.Categories[0].Entries)
	stagingCountBefore := len(m.hostsFile.Categories[1].Entries)

	// Execute move with Enter key
	newModel, _ := m.updateMove(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)

	// Should return to main view
	if m.currentView != viewMain {
		t.Errorf("Expected view to return to main after move execution")
	}

	// Check that move was executed
	devCountAfter := len(m.hostsFile.Categories[0].Entries)
	stagingCountAfter := len(m.hostsFile.Categories[1].Entries)

	if devCountAfter != devCountBefore-1 {
		t.Errorf("Expected development category to lose 1 entry")
	}
	if stagingCountAfter != stagingCountBefore+1 {
		t.Errorf("Expected staging category to gain 1 entry")
	}

	// Check success message
	if !contains(m.message, "Moved dev.local from development to staging") {
		t.Errorf("Expected success message, got: %s", m.message)
	}
}

func TestMainViewMoveActivation(t *testing.T) {
	m := createTestModel()

	// Test activating move mode with 'm' key
	m.cursor = 1 // Select second entry
	newModel, _ := m.updateMain(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = newModel.(*model)

	// Should switch to move view
	if m.currentView != viewMove {
		t.Errorf("Expected view to switch to move mode")
	}

	// Should set up move state correctly
	if m.moveEntryIndex != 1 {
		t.Errorf("Expected moveEntryIndex to be 1, got %d", m.moveEntryIndex)
	}

	if m.moveCategoryCursor != 0 {
		t.Errorf("Expected moveCategoryCursor to be 0, got %d", m.moveCategoryCursor)
	}

	// Should set a valid target category (not the current one)
	currentCategory := m.entries[1].category
	if m.moveTargetCategory == "" || m.moveTargetCategory == currentCategory {
		t.Errorf("Expected valid target category different from current (%s), got %s",
			currentCategory, m.moveTargetCategory)
	}
}

func TestMainViewMoveActivationNoEntry(t *testing.T) {
	m := createTestModel()

	// Test activating move mode when cursor is beyond entries
	m.cursor = 999
	oldView := m.currentView
	newModel, _ := m.updateMain(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = newModel.(*model)

	// Should not switch views
	if m.currentView != oldView {
		t.Errorf("Expected view to remain the same when no entry is selected")
	}

	// Should show error message
	if !contains(m.message, "No entry selected to move") {
		t.Errorf("Expected error message for no entry selected, got: %s", m.message)
	}
}

func TestViewMoveRendering(t *testing.T) {
	m := createTestModel()

	// Set up move mode
	m.currentView = viewMove
	m.moveEntryIndex = 0
	m.moveCategoryCursor = 0
	m.moveTargetCategory = "staging"

	// Test view rendering
	output := m.viewMove()

	// Should contain title
	if !contains(output, "Move Entry to Category") {
		t.Errorf("Expected move view title in output")
	}

	// Should show entry being moved
	if !contains(output, "Moving: 127.0.0.1") {
		t.Errorf("Expected entry details in move view")
	}
	if !contains(output, "dev.local") {
		t.Errorf("Expected hostname in move view")
	}
	if !contains(output, "From category: development") {
		t.Errorf("Expected source category in move view")
	}

	// Should show available categories
	if !contains(output, "staging") {
		t.Errorf("Expected staging category option in move view")
	}
	if !contains(output, "production") {
		t.Errorf("Expected production category option in move view")
	}

	// Should show instructions
	if !contains(output, "Use ↑/↓ to select category") {
		t.Errorf("Expected navigation instructions in move view")
	}
}

func TestMultipleMovesIntegrity(t *testing.T) {
	m := createTestModel()

	// Perform multiple moves to test data integrity
	moves := []struct {
		entryIndex     int
		targetCategory string
	}{
		{0, "staging"},     // dev.local: development -> staging
		{1, "production"},  // api.dev: development -> production
		{0, "development"}, // staging.local: staging -> development
	}

	for i, move := range moves {
		// Find current position of entry after previous moves
		m.entries = buildEntryList(m.hostsFile)

		if move.entryIndex >= len(m.entries) {
			t.Fatalf("Move %d: entry index %d out of bounds", i, move.entryIndex)
		}

		entryBefore := m.entries[move.entryIndex]

		// Perform move
		err := m.moveEntry(move.entryIndex, move.targetCategory)
		if err != nil {
			t.Fatalf("Move %d failed: %v", i, err)
		}

		// Verify integrity
		totalEntriesAfter := 0
		for _, category := range m.hostsFile.Categories {
			totalEntriesAfter += len(category.Entries)
		}

		if totalEntriesAfter != 4 {
			t.Errorf("Move %d: total entries changed from 4 to %d", i, totalEntriesAfter)
		}

		// Verify entry exists in target category
		found := false
		for _, category := range m.hostsFile.Categories {
			if category.Name == move.targetCategory {
				for _, entry := range category.Entries {
					if len(entry.Hostnames) > 0 && len(entryBefore.entry.Hostnames) > 0 &&
						entry.Hostnames[0] == entryBefore.entry.Hostnames[0] &&
						entry.Category == move.targetCategory {
						found = true
						break
					}
				}
				break
			}
		}

		if !found {
			t.Errorf("Move %d: entry not found in target category %s", i, move.targetCategory)
		}
	}
}

func TestValidateCategoryName(t *testing.T) {
	m := createTestModel()

	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"valid name", "testing", false},
		{"valid with underscore", "test_category", false},
		{"valid with dash", "test-category", false},
		{"valid with numbers", "test123", false},
		{"valid mixed", "Test_Category-123", false},
		{"empty name", "", true},
		{"too long", strings.Repeat("a", 51), true},
		{"invalid characters space", "test category", true},
		{"invalid characters dot", "test.category", true},
		{"invalid characters slash", "test/category", true},
		{"invalid characters special", "test@category", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.validateCategoryName(tt.input)
			if tt.expectErr && err == nil {
				t.Errorf("Expected error for input '%s', got nil", tt.input)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error for input '%s', got: %v", tt.input, err)
			}
		})
	}
}

func TestCreateCategory(t *testing.T) {
	m := createTestModel()

	initialCount := len(m.hostsFile.Categories)

	// Create a new category
	err := m.createCategory("testing", "Test category description")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	// Check that category was added
	if len(m.hostsFile.Categories) != initialCount+1 {
		t.Errorf("Expected %d categories, got %d", initialCount+1, len(m.hostsFile.Categories))
	}

	// Find the created category
	var createdCategory *hosts.Category
	for i := range m.hostsFile.Categories {
		if m.hostsFile.Categories[i].Name == "testing" {
			createdCategory = &m.hostsFile.Categories[i]
			break
		}
	}

	if createdCategory == nil {
		t.Errorf("Created category 'testing' not found")
		return
	}

	// Verify category properties
	if createdCategory.Description != "Test category description" {
		t.Errorf("Expected description 'Test category description', got '%s'", createdCategory.Description)
	}
	if !createdCategory.Enabled {
		t.Errorf("Expected category to be enabled")
	}
	if len(createdCategory.Entries) != 0 {
		t.Errorf("Expected empty entries list, got %d entries", len(createdCategory.Entries))
	}
}

func TestUpdateCreateCategoryNavigation(t *testing.T) {
	m := createTestModel()

	// Set up create category mode
	m.currentView = viewCreateCategory
	m.createCategoryField = 0
	m.createCategoryName = ""
	m.createCategoryDescription = ""

	// Test tab navigation (forward)
	newModel, _ := m.updateCreateCategory(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(*model)

	if m.createCategoryField != 1 {
		t.Errorf("Expected field to advance to 1, got %d", m.createCategoryField)
	}

	// Test shift+tab navigation (forward - same as tab in this implementation)
	newModel, _ = m.updateCreateCategory(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = newModel.(*model)

	if m.createCategoryField != 0 {
		t.Errorf("Expected field to wrap back to 0, got %d", m.createCategoryField)
	}

	// Test escape (cancel)
	newModel, _ = m.updateCreateCategory(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(*model)

	if m.currentView != viewMain {
		t.Errorf("Expected view to return to main after escape")
	}
}

func TestUpdateCreateCategoryInput(t *testing.T) {
	m := createTestModel()

	// Set up create category mode
	m.currentView = viewCreateCategory
	m.createCategoryField = 0

	// Test character input for name field
	newModel, _ := m.updateCreateCategory(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = newModel.(*model)
	if m.createCategoryName != "t" {
		t.Errorf("Expected name to be 't', got '%s'", m.createCategoryName)
	}

	newModel, _ = m.updateCreateCategory(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e', 's', 't'}})
	m = newModel.(*model)
	// Note: multiple runes in single message may not work as expected in real usage
	// but we can test single character addition
	for _, r := range []rune{'e', 's', 't'} {
		newModel, _ = m.updateCreateCategory(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(*model)
	}

	expectedName := "test"
	if m.createCategoryName != expectedName {
		t.Errorf("Expected name to be '%s', got '%s'", expectedName, m.createCategoryName)
	}

	// Test backspace
	newModel, _ = m.updateCreateCategory(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(*model)
	if m.createCategoryName != "tes" {
		t.Errorf("Expected name to be 'tes' after backspace, got '%s'", m.createCategoryName)
	}

	// Switch to description field and test input
	m.createCategoryField = 1
	newModel, _ = m.updateCreateCategory(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newModel.(*model)
	if m.createCategoryDescription != "d" {
		t.Errorf("Expected description to be 'd', got '%s'", m.createCategoryDescription)
	}
}

func TestUpdateCreateCategoryExecution(t *testing.T) {
	m := createTestModel()

	// Set up create category mode
	m.currentView = viewCreateCategory
	m.createCategoryName = "testing"
	m.createCategoryDescription = "Test category"
	m.createCategoryField = 0

	initialCount := len(m.hostsFile.Categories)
	initialCatCount := len(m.categories)

	// Execute category creation with Enter key
	newModel, _ := m.updateCreateCategory(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)

	// Should return to main view
	if m.currentView != viewMain {
		t.Errorf("Expected view to return to main after category creation")
	}

	// Check that category was created
	if len(m.hostsFile.Categories) != initialCount+1 {
		t.Errorf("Expected %d categories in hosts file, got %d", initialCount+1, len(m.hostsFile.Categories))
	}

	// Check that categories list was updated
	if len(m.categories) != initialCatCount+1 {
		t.Errorf("Expected %d categories in list, got %d", initialCatCount+1, len(m.categories))
	}

	// Check success message
	if !contains(m.message, "Created category: testing") {
		t.Errorf("Expected success message, got: %s", m.message)
	}
}

func TestUpdateCreateCategoryValidation(t *testing.T) {
	m := createTestModel()

	// Test empty name
	m.currentView = viewCreateCategory
	m.createCategoryName = ""
	m.createCategoryDescription = "Description"

	newModel, _ := m.updateCreateCategory(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)

	// Should stay in create view with error message
	if m.currentView != viewCreateCategory {
		t.Errorf("Expected to stay in create category view for empty name")
	}
	if !contains(m.message, "Category name is required") {
		t.Errorf("Expected name required error, got: %s", m.message)
	}

	// Test invalid name
	m.createCategoryName = "invalid name with spaces"
	m.message = ""

	newModel, _ = m.updateCreateCategory(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)

	if m.currentView != viewCreateCategory {
		t.Errorf("Expected to stay in create category view for invalid name")
	}
	if !contains(m.message, "Invalid category name") {
		t.Errorf("Expected invalid name error, got: %s", m.message)
	}

	// Test duplicate name
	m.createCategoryName = "development" // Already exists in test data
	m.message = ""

	newModel, _ = m.updateCreateCategory(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)

	if m.currentView != viewCreateCategory {
		t.Errorf("Expected to stay in create category view for duplicate name")
	}
	if !contains(m.message, "already exists") {
		t.Errorf("Expected duplicate name error, got: %s", m.message)
	}
}

func TestMainViewCreateCategoryActivation(t *testing.T) {
	m := createTestModel()

	// Test activating create category mode with 'c' key
	newModel, _ := m.updateMain(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = newModel.(*model)

	// Should switch to create category view
	if m.currentView != viewCreateCategory {
		t.Errorf("Expected view to switch to create category mode")
	}

	// Should initialize create category state
	if m.createCategoryName != "" {
		t.Errorf("Expected empty category name, got '%s'", m.createCategoryName)
	}
	if m.createCategoryDescription != "" {
		t.Errorf("Expected empty category description, got '%s'", m.createCategoryDescription)
	}
	if m.createCategoryField != 0 {
		t.Errorf("Expected field to be 0, got %d", m.createCategoryField)
	}
}

func TestViewCreateCategoryRendering(t *testing.T) {
	m := createTestModel()

	// Set up create category mode with some data
	m.currentView = viewCreateCategory
	m.createCategoryName = "test-category"
	m.createCategoryDescription = "Test description"
	m.createCategoryField = 0

	// Test view rendering
	output := m.viewCreateCategory()

	// Should contain title
	if !contains(output, "Create New Category") {
		t.Errorf("Expected create category title in output")
	}

	// Should show category name
	if !contains(output, "test-category") {
		t.Errorf("Expected category name in output")
	}

	// Should show description
	if !contains(output, "Test description") {
		t.Errorf("Expected description in output")
	}

	// Should show field labels
	if !contains(output, "Category Name:") {
		t.Errorf("Expected name label in output")
	}
	if !contains(output, "Description (optional):") {
		t.Errorf("Expected description label in output")
	}

	// Should show validation rules
	if !contains(output, "a-z, A-Z, 0-9, _, -") {
		t.Errorf("Expected validation rules in output")
	}

	// Should show instructions
	if !contains(output, "Use Tab/Shift+Tab to navigate") {
		t.Errorf("Expected navigation instructions")
	}
	if !contains(output, "Press Enter to create category") {
		t.Errorf("Expected creation instructions")
	}
}

func TestCategoryIntegrationWithMove(t *testing.T) {
	m := createTestModel()

	// Create a new category
	err := m.createCategory("integration-test", "Integration test category")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	// Update categories list and entries (simulating what the TUI would do)
	m.categories = append(m.categories, "integration-test")
	m.entries = buildEntryList(m.hostsFile)

	// Test that the new category appears in available categories for move
	m.moveEntryIndex = 0 // First entry (development category)
	availableCategories := m.getAvailableCategories()

	found := false
	for _, cat := range availableCategories {
		if cat == "integration-test" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("New category should be available for move operations")
	}

	// Test moving an entry to the new category
	initialCount := len(m.hostsFile.Categories[len(m.hostsFile.Categories)-1].Entries)
	err = m.moveEntry(0, "integration-test")
	if err != nil {
		t.Errorf("Failed to move entry to new category: %v", err)
	}

	// Check that the entry was moved to the new category
	newCount := len(m.hostsFile.Categories[len(m.hostsFile.Categories)-1].Entries)
	if newCount != initialCount+1 {
		t.Errorf("Expected new category to have %d entries, got %d", initialCount+1, newCount)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsHelper(s, substr))))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestEditEntryActivation(t *testing.T) {
	m := createTestModel()
	m.cursor = 0 // Select first entry

	// Test activating edit mode with 'e' key
	newModel, _ := m.updateMain(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = newModel.(*model)

	if m.currentView != viewEdit {
		t.Errorf("Expected current view to be viewEdit, got %v", m.currentView)
	}

	if m.editEntryIndex != 0 {
		t.Errorf("Expected editEntryIndex to be 0, got %d", m.editEntryIndex)
	}

	// Check that edit fields are populated with current entry data
	expectedEntry := m.entries[0].entry
	if m.editIP != expectedEntry.IP {
		t.Errorf("Expected editIP to be '%s', got '%s'", expectedEntry.IP, m.editIP)
	}
	if m.editHostnames != strings.Join(expectedEntry.Hostnames, " ") {
		t.Errorf("Expected editHostnames to be '%s', got '%s'", strings.Join(expectedEntry.Hostnames, " "), m.editHostnames)
	}
	if m.editComment != expectedEntry.Comment {
		t.Errorf("Expected editComment to be '%s', got '%s'", expectedEntry.Comment, m.editComment)
	}
	if m.editCategory != expectedEntry.Category {
		t.Errorf("Expected editCategory to be '%s', got '%s'", expectedEntry.Category, m.editCategory)
	}
	if m.editField != 0 {
		t.Errorf("Expected editField to be 0, got %d", m.editField)
	}
}

func TestEditEntryActivationNoSelection(t *testing.T) {
	m := createTestModel()
	m.entries = []entryWithIndex{} // No entries

	// Test activating edit mode with no entries
	newModel, _ := m.updateMain(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = newModel.(*model)

	if m.currentView != viewMain {
		t.Errorf("Expected current view to remain viewMain when no entries, got %v", m.currentView)
	}

	if m.message != "No entry selected to edit" {
		t.Errorf("Expected message 'No entry selected to edit', got '%s'", m.message)
	}
}

func TestUpdateEditNavigation(t *testing.T) {
	m := createTestModel()
	m.currentView = viewEdit
	m.editField = 0

	// Test Tab navigation
	newModel, _ := m.updateEdit(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(*model)
	if m.editField != 1 {
		t.Errorf("Expected editField to be 1 after Tab, got %d", m.editField)
	}

	// Test Shift+Tab navigation
	newModel, _ = m.updateEdit(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = newModel.(*model)
	if m.editField != 0 {
		t.Errorf("Expected editField to be 0 after Shift+Tab, got %d", m.editField)
	}

	// Test navigation wrap-around with Tab
	m.editField = 3
	newModel, _ = m.updateEdit(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(*model)
	if m.editField != 0 {
		t.Errorf("Expected editField to wrap to 0, got %d", m.editField)
	}

	// Test Escape to cancel
	newModel, _ = m.updateEdit(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(*model)
	if m.currentView != viewMain {
		t.Errorf("Expected current view to be viewMain after Esc, got %v", m.currentView)
	}
}

func TestUpdateEditInput(t *testing.T) {
	m := createTestModel()
	m.currentView = viewEdit
	m.editIP = ""
	m.editHostnames = ""
	m.editComment = ""
	m.editCategory = ""

	// Test IP input (field 0)
	m.editField = 0
	newModel, _ := m.updateEdit(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = newModel.(*model)
	if m.editIP != "1" {
		t.Errorf("Expected editIP to be '1', got '%s'", m.editIP)
	}

	// Test hostname input (field 1) - input characters one by one
	m.editField = 1
	for _, char := range "test" {
		newModel, _ = m.updateEdit(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		m = newModel.(*model)
	}
	if m.editHostnames != "test" {
		t.Errorf("Expected editHostnames to be 'test', got '%s'", m.editHostnames)
	}

	// Test comment input (field 2) - input characters one by one
	m.editField = 2
	for _, char := range "comment" {
		newModel, _ = m.updateEdit(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		m = newModel.(*model)
	}
	if m.editComment != "comment" {
		t.Errorf("Expected editComment to be 'comment', got '%s'", m.editComment)
	}

	// Test category input (field 3) - input characters one by one
	m.editField = 3
	for _, char := range "dev" {
		newModel, _ = m.updateEdit(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		m = newModel.(*model)
	}
	if m.editCategory != "dev" {
		t.Errorf("Expected editCategory to be 'dev', got '%s'", m.editCategory)
	}

	// Test backspace
	newModel, _ = m.updateEdit(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(*model)
	if m.editCategory != "de" {
		t.Errorf("Expected editCategory to be 'de' after backspace, got '%s'", m.editCategory)
	}
}

func TestUpdateEditExecution(t *testing.T) {
	m := createTestModel()
	m.currentView = viewEdit
	m.editEntryIndex = 0
	m.editIP = "10.0.0.1"
	m.editHostnames = "new.example.com"
	m.editComment = "Updated comment"
	m.editCategory = "development"

	// Test successful edit
	newModel, _ := m.updateEdit(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)

	if m.currentView != viewMain {
		t.Errorf("Expected current view to be viewMain after successful edit, got %v", m.currentView)
	}

	if m.message != "Entry updated successfully" {
		t.Errorf("Expected success message, got '%s'", m.message)
	}

	// Check that entry was actually updated
	updatedEntry := m.entries[0].entry
	if updatedEntry.IP != "10.0.0.1" {
		t.Errorf("Expected updated IP to be '10.0.0.1', got '%s'", updatedEntry.IP)
	}
	if len(updatedEntry.Hostnames) != 1 || updatedEntry.Hostnames[0] != "new.example.com" {
		t.Errorf("Expected updated hostnames to be ['new.example.com'], got %v", updatedEntry.Hostnames)
	}
	if updatedEntry.Comment != "Updated comment" {
		t.Errorf("Expected updated comment to be 'Updated comment', got '%s'", updatedEntry.Comment)
	}
}

func TestUpdateEditValidation(t *testing.T) {
	m := createTestModel()
	m.currentView = viewEdit
	m.editEntryIndex = 0

	// Test validation with empty IP
	m.editIP = ""
	m.editHostnames = "test.com"
	m.editComment = ""
	m.editCategory = "development"

	newModel, _ := m.updateEdit(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)

	if m.currentView != viewEdit {
		t.Errorf("Expected to remain in viewEdit after validation error, got %v", m.currentView)
	}
	if m.message != "IP and hostnames are required" {
		t.Errorf("Expected validation message, got '%s'", m.message)
	}

	// Test validation with empty hostnames
	m.editIP = "10.0.0.1"
	m.editHostnames = ""
	m.message = "" // Clear previous message

	newModel, _ = m.updateEdit(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)

	if m.currentView != viewEdit {
		t.Errorf("Expected to remain in viewEdit after validation error, got %v", m.currentView)
	}
	if m.message != "IP and hostnames are required" {
		t.Errorf("Expected validation message, got '%s'", m.message)
	}

	// Test validation with whitespace-only hostnames
	m.editHostnames = "   "
	m.message = "" // Clear previous message

	newModel, _ = m.updateEdit(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(*model)

	if m.currentView != viewEdit {
		t.Errorf("Expected to remain in viewEdit after validation error, got %v", m.currentView)
	}
	if m.message != "At least one hostname is required" {
		t.Errorf("Expected hostname validation message, got '%s'", m.message)
	}
}

func TestViewEditRendering(t *testing.T) {
	m := createTestModel()
	m.currentView = viewEdit
	m.editIP = "10.0.0.1"
	m.editHostnames = "test.example.com"
	m.editComment = "Test comment"
	m.editCategory = "development"
	m.editField = 1 // Select hostnames field

	output := m.viewEdit()

	// Check that the view contains expected content
	if !strings.Contains(output, "Edit Entry") {
		t.Errorf("Expected output to contain title 'Edit Entry'")
	}
	if !strings.Contains(output, "10.0.0.1") {
		t.Errorf("Expected output to contain IP '10.0.0.1'")
	}
	if !strings.Contains(output, "test.example.com") {
		t.Errorf("Expected output to contain hostname 'test.example.com'")
	}
	if !strings.Contains(output, "Test comment") {
		t.Errorf("Expected output to contain comment 'Test comment'")
	}
	if !strings.Contains(output, "development") {
		t.Errorf("Expected output to contain category 'development'")
	}
	if !strings.Contains(output, "Tab/Shift+Tab") {
		t.Errorf("Expected output to contain navigation instructions")
	}
	if !strings.Contains(output, "Enter to save") {
		t.Errorf("Expected output to contain save instructions")
	}
}
