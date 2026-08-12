package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brandonhon/hosts-manager/internal/hosts"
)

func TestCategoryAddCmd(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Add category without arguments",
			args:          []string{},
			expectError:   true,
			errorContains: "accepts between 1 and 2 arg(s), received 0",
		},
		{
			name:          "Add category with too many arguments",
			args:          []string{"testing", "description", "extra"},
			expectError:   true,
			errorContains: "accepts between 1 and 2 arg(s), received 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the command
			cmd := categoryAddCmd()

			// Capture output
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)

			// Set the command args
			cmd.SetArgs(tt.args)

			// Execute the command - we only test argument validation here
			// since the actual execution requires elevated privileges
			err := cmd.Execute()

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}
		})
	}
}

func TestCategoryAddCmdStructure(t *testing.T) {
	// Test that the command is properly structured
	cmd := categoryAddCmd()

	if cmd.Use != "add <name> [description]" {
		t.Errorf("Expected Use to be 'add <name> [description]', got: %s", cmd.Use)
	}

	if cmd.Short != "Add a new category" {
		t.Errorf("Expected Short to be 'Add a new category', got: %s", cmd.Short)
	}

	// Test that command has proper argument validation
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	// Test with no arguments
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error with no arguments")
	} else if !strings.Contains(err.Error(), "accepts between 1 and 2 arg(s), received 0") {
		t.Errorf("Expected specific argument error, got: %v", err)
	}

	// Test with too many arguments
	cmd.SetArgs([]string{"cat1", "desc1", "extra"})
	err = cmd.Execute()
	if err == nil {
		t.Error("Expected error with too many arguments")
	} else if !strings.Contains(err.Error(), "accepts between 1 and 2 arg(s), received 3") {
		t.Errorf("Expected specific argument error, got: %v", err)
	}
}

// TestAddEntry tests adding entries to hosts file
func TestAddEntry(t *testing.T) {
	tests := []struct {
		name         string
		ip           string
		hostnames    []string
		category     string
		comment      string
		wantErr      bool
		skipValidate bool
	}{
		{
			name:      "add single hostname",
			ip:        "192.168.1.100",
			hostnames: []string{"test.local"},
			category:  "development",
			wantErr:   false,
		},
		{
			name:      "add multiple hostnames",
			ip:        "192.168.1.101",
			hostnames: []string{"api.local", "app.local"},
			category:  "development",
			wantErr:   false,
		},
		{
			name:      "add with comment",
			ip:        "192.168.1.102",
			hostnames: []string{"db.local"},
			category:  "development",
			comment:   "Database server",
			wantErr:   false,
		},
		{
			name:         "add invalid IP",
			ip:           "999.999.999.999",
			hostnames:    []string{"invalid.local"},
			category:     "development",
			wantErr:      true,
			skipValidate: true,
		},
		{
			name:         "add invalid hostname",
			ip:           "192.168.1.103",
			hostnames:    []string{"invalid hostname with spaces"},
			category:     "development",
			wantErr:      true,
			skipValidate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipValidate && testing.Short() {
				t.Skip("Skipping validation test in short mode")
			}

			ctx := setupTestEnvironment(t)

			// Perform add operation
			hf := ctx.ParseHostsFile()

			entry := hosts.Entry{
				IP:        tt.ip,
				Hostnames: tt.hostnames,
				Comment:   tt.comment,
				Category:  tt.category,
				Enabled:   true,
			}

			// Validate before adding
			if err := hosts.ValidateIP(tt.ip); err != nil {
				if !tt.wantErr {
					t.Errorf("ValidateIP() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			for _, hostname := range tt.hostnames {
				if err := hosts.ValidateHostname(hostname); err != nil {
					if !tt.wantErr {
						t.Errorf("ValidateHostname() error = %v, wantErr %v", err, tt.wantErr)
					}
					return
				}
			}

			err := hf.AddEntry(entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Save and verify
			ctx.WriteHostsFileFromStruct(hf)

			// Verify entry exists
			ctx.AssertEntryExists(tt.ip, tt.hostnames[0])
		})
	}
}

// TestDeleteEntry tests deleting entries from hosts file
func TestDeleteEntry(t *testing.T) {
	ctx := setupTestEnvironment(t)

	// Verify initial state
	ctx.AssertEntryExists("127.0.0.1", "dev.local")

	// Delete entry
	hf := ctx.ParseHostsFile()
	if !hf.RemoveEntry("dev.local") {
		t.Fatal("RemoveEntry() = false, want true")
	}

	ctx.WriteHostsFileFromStruct(hf)

	// Verify entry removed
	ctx.AssertEntryNotExists("127.0.0.1", "dev.local")
}

// TestEnableDisableEntries tests enabling and disabling entries
func TestEnableDisableEntries(t *testing.T) {
	ctx := setupTestEnvironment(t)

	// Parse hosts file
	hf := ctx.ParseHostsFile()

	// Find and disable an entry
	var found bool
	for _, cat := range hf.Categories {
		for i := range cat.Entries {
			if cat.Entries[i].IP == "127.0.0.1" && containsHostname(cat.Entries[i].Hostnames, "dev.local") {
				cat.Entries[i].Enabled = false
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		t.Fatal("Could not find entry to disable")
	}

	ctx.WriteHostsFileFromStruct(hf)
	ctx.AssertEntryDisabled("127.0.0.1", "dev.local")

	// Re-enable the entry
	hf = ctx.ParseHostsFile()
	found = false
	for _, cat := range hf.Categories {
		for i := range cat.Entries {
			if cat.Entries[i].IP == "127.0.0.1" && containsHostname(cat.Entries[i].Hostnames, "dev.local") {
				cat.Entries[i].Enabled = true
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		t.Fatal("Could not find entry to enable")
	}

	ctx.WriteHostsFileFromStruct(hf)
	ctx.AssertEntryEnabled("127.0.0.1", "dev.local")
}

// TestCategoryOperations tests category-related operations
func TestCategoryOperations(t *testing.T) {
	t.Run("add category", func(t *testing.T) {
		ctx := setupTestEnvironment(t)

		hf := ctx.ParseHostsFile()

		// Add new category
		err := hf.AddCategory("testing", "Testing environment")
		if err != nil {
			t.Fatalf("AddCategory() error = %v", err)
		}

		// Verify category exists
		found := false
		for _, cat := range hf.Categories {
			if cat.Name == "testing" {
				found = true
				if cat.Description != "Testing environment" {
					t.Errorf("Category description = %q, want %q", cat.Description, "Testing environment")
				}
				break
			}
		}

		if !found {
			t.Error("Category 'testing' not found after adding")
		}
	})

	t.Run("list categories", func(t *testing.T) {
		ctx := setupTestEnvironment(t)

		hf := ctx.ParseHostsFile()

		// Should have at least: development, staging, custom
		if len(hf.Categories) < 3 {
			t.Errorf("Expected at least 3 categories, got %d", len(hf.Categories))
		}

		// Check for specific categories
		categoryNames := make([]string, len(hf.Categories))
		for i, cat := range hf.Categories {
			categoryNames[i] = cat.Name
		}

		expectedCategories := []string{"development", "staging", "custom"}
		for _, expected := range expectedCategories {
			if !containsStr(categoryNames, expected) {
				t.Errorf("Expected category %q not found. Got: %v", expected, categoryNames)
			}
		}
	})

	t.Run("disable category", func(t *testing.T) {
		ctx := setupTestEnvironment(t)

		hf := ctx.ParseHostsFile()

		// Disable development category
		var developmentCat *hosts.Category
		for i := range hf.Categories {
			if hf.Categories[i].Name == "development" {
				developmentCat = &hf.Categories[i]
				break
			}
		}

		if developmentCat == nil {
			t.Fatal("Development category not found")
		}

		// Disable all entries in the category
		for i := range developmentCat.Entries {
			developmentCat.Entries[i].Enabled = false
		}

		ctx.WriteHostsFileFromStruct(hf)

		// Verify entries are disabled
		ctx.AssertEntryDisabled("127.0.0.1", "dev.local")
		ctx.AssertEntryDisabled("192.168.1.100", "api.dev")
	})
}

// TestSearch tests the search functionality
func TestSearch(t *testing.T) {
	ctx := setupTestEnvironment(t)

	hf := ctx.ParseHostsFile()

	tests := []struct {
		name          string
		query         string
		wantMatches   int
		wantHostnames []string
	}{
		{
			name:          "search by hostname",
			query:         "dev",
			wantMatches:   2, // dev.local, api.dev
			wantHostnames: []string{"dev.local", "api.dev"},
		},
		{
			name:          "search by IP",
			query:         "127.0.0.1",
			wantMatches:   2, // localhost, dev.local
			wantHostnames: []string{"localhost", "dev.local"},
		},
		{
			name:          "search by category",
			query:         "development",
			wantMatches:   2,
			wantHostnames: []string{"dev.local", "api.dev"},
		},
		{
			name:        "no matches",
			query:       "nonexistent",
			wantMatches: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := searchHostsFile(hf, tt.query)

			if len(matches) != tt.wantMatches {
				t.Errorf("Search returned %d matches, want %d", len(matches), tt.wantMatches)
			}

			if tt.wantHostnames != nil {
				for _, wantHostname := range tt.wantHostnames {
					found := false
					for _, match := range matches {
						if containsHostname(match.Hostnames, wantHostname) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected hostname %q not found in search results", wantHostname)
					}
				}
			}
		})
	}
}

// TestBackupRestore tests backup and restore functionality
func TestBackupRestore(t *testing.T) {
	ctx := setupTestEnvironment(t)

	// Get initial content
	initialContent := ctx.ReadHostsFile()

	// Create backup
	backupFile := filepath.Join(ctx.BackupDir, "test-backup.hosts")
	if err := os.WriteFile(backupFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	ctx.AssertBackupExists("test-backup.hosts")

	// Modify hosts file
	hf := ctx.ParseHostsFile()
	entry := hosts.Entry{
		IP:        "10.0.0.1",
		Hostnames: []string{"newhost.local"},
		Category:  "development",
		Enabled:   true,
	}
	if err := hf.AddEntry(entry); err != nil {
		t.Fatalf("AddEntry() error = %v", err)
	}
	ctx.WriteHostsFileFromStruct(hf)

	// Verify modification
	ctx.AssertEntryExists("10.0.0.1", "newhost.local")

	// Restore from backup
	backupContent, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}

	ctx.WriteHostsFile(string(backupContent))

	// Verify restoration
	restoredContent := ctx.ReadHostsFile()
	if restoredContent != initialContent {
		t.Error("Restored content does not match initial content")
	}

	ctx.AssertEntryNotExists("10.0.0.1", "newhost.local")
}

// TestImportExport tests import and export functionality
func TestImportExport(t *testing.T) {
	ctx := setupTestEnvironment(t)

	// Export current hosts file
	exportFile := filepath.Join(ctx.TempDir, "export.yaml")
	hf := ctx.ParseHostsFile()

	// Create export content (simplified YAML)
	exportContent := "# Hosts file export\n"
	exportContent += "categories:\n"
	for _, cat := range hf.Categories {
		exportContent += "  - name: " + cat.Name + "\n"
		exportContent += "    entries:\n"
		for _, entry := range cat.Entries {
			exportContent += "      - ip: " + entry.IP + "\n"
			exportContent += "        hostnames: [" + strings.Join(entry.Hostnames, ", ") + "]\n"
		}
	}

	if err := os.WriteFile(exportFile, []byte(exportContent), 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	// Verify export file exists
	if _, err := os.Stat(exportFile); os.IsNotExist(err) {
		t.Error("Export file was not created")
	}
}

// TestConfigOperations tests configuration operations
func TestConfigOperations(t *testing.T) {
	ctx := setupTestEnvironment(t)

	t.Run("load config", func(t *testing.T) {
		cfg := ctx.LoadConfig()

		if cfg == nil {
			t.Fatal("Config should not be nil")
		}

		// Check default categories exist
		if len(cfg.Categories) == 0 {
			t.Error("Default categories should not be empty")
		}
	})

	t.Run("modify config", func(t *testing.T) {
		cfg := ctx.LoadConfig()

		// Add custom category to config
		cfg.Categories["custom-test"] = "Custom test category"

		ctx.WriteConfig(cfg)

		// Reload and verify
		reloaded := ctx.LoadConfig()
		if _, found := reloaded.Categories["custom-test"]; !found {
			t.Error("Custom category not found in reloaded config")
		}
	})
}

// Helper function to search hosts file (simplified)
func searchHostsFile(hf *hosts.HostsFile, query string) []hosts.Entry {
	var matches []hosts.Entry
	query = strings.ToLower(query)

	for _, cat := range hf.Categories {
		// Check category name
		if strings.Contains(strings.ToLower(cat.Name), query) {
			matches = append(matches, cat.Entries...)
			continue
		}

		// Check entries
		for _, entry := range cat.Entries {
			// Check IP
			if strings.Contains(strings.ToLower(entry.IP), query) {
				matches = append(matches, entry)
				continue
			}

			// Check hostnames
			for _, hostname := range entry.Hostnames {
				if strings.Contains(strings.ToLower(hostname), query) {
					matches = append(matches, entry)
					break
				}
			}
		}
	}

	return matches
}

// Helper functions
func containsHostname(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestExportToHostsRoundTrip guards the category loss: exportToHosts used to
// emit only "# ==== NAME ====" banners, which the parser treats as decoration,
// so re-reading an exported hosts file put every entry in "default".
func TestExportToHostsRoundTrip(t *testing.T) {
	hf := &hosts.HostsFile{Categories: []hosts.Category{
		{Name: "development", Description: "Dev stuff", Enabled: true, Entries: []hosts.Entry{
			{IP: "127.0.0.1", Hostnames: []string{"dev.local"}, Category: "development", Enabled: true},
		}},
		{Name: "blocked", Enabled: true, Entries: []hosts.Entry{
			{IP: "10.0.0.9", Hostnames: []string{"ads.example.com"}, Category: "blocked", Enabled: true},
		}},
	}}

	data, err := exportToHosts(hf)
	if err != nil {
		t.Fatalf("exportToHosts() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "hosts")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	back, err := hosts.NewParser(path).Parse()
	if err != nil {
		t.Fatalf("re-parse error = %v", err)
	}

	got := map[string]int{}
	for _, c := range back.Categories {
		got[c.Name] = len(c.Entries)
	}
	for _, want := range []string{"development", "blocked"} {
		if got[want] != 1 {
			t.Errorf("category %q has %d entries after round trip, want 1 (all categories: %v)\n%s",
				want, got[want], got, data)
		}
	}
	if n := got["default"]; n != 0 {
		t.Errorf("%d entries fell into \"default\"; category markers were not preserved", n)
	}

	// The description should survive too, since it rides on the @category line.
	if c := back.GetCategory("development"); c == nil || c.Description != "Dev stuff" {
		t.Errorf("description lost: %+v", c)
	}
}

// TestValidateImportedHosts covers the replace path of `import`, which wrote
// straight to the hosts file without validating anything.
func TestValidateImportedHosts(t *testing.T) {
	tests := []struct {
		name      string
		hf        *hosts.HostsFile
		expectErr bool
	}{
		{
			name: "valid",
			hf: &hosts.HostsFile{Categories: []hosts.Category{{Name: "custom", Entries: []hosts.Entry{
				{IP: "127.0.0.1", Hostnames: []string{"ok.local"}, Category: "custom", Enabled: true}}}}},
		},
		{
			name: "comment injects a host mapping",
			hf: &hosts.HostsFile{Categories: []hosts.Category{{Name: "custom", Entries: []hosts.Entry{
				{IP: "127.0.0.1", Hostnames: []string{"safe.local"}, Category: "custom", Enabled: true,
					Comment: "note\n10.0.0.1 evil.example.com"}}}}},
			expectErr: true,
		},
		{
			name: "invalid ip",
			hf: &hosts.HostsFile{Categories: []hosts.Category{{Name: "custom", Entries: []hosts.Entry{
				{IP: "999.999.999.999", Hostnames: []string{"bad.local"}, Category: "custom"}}}}},
			expectErr: true,
		},
		{
			name: "invalid hostname",
			hf: &hosts.HostsFile{Categories: []hosts.Category{{Name: "custom", Entries: []hosts.Entry{
				{IP: "127.0.0.1", Hostnames: []string{"has space.local"}, Category: "custom"}}}}},
			expectErr: true,
		},
		{
			name: "category name taken from the enclosing category when the entry omits it",
			hf: &hosts.HostsFile{Categories: []hosts.Category{{Name: "bad category!", Entries: []hosts.Entry{
				{IP: "127.0.0.1", Hostnames: []string{"ok.local"}}}}}},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateImportedHosts(tt.hf)
			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
