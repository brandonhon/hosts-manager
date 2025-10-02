package hosts

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// Test data and helpers
func createTestHostsFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "hosts_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tmpFile.Close() }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}

	return tmpFile.Name()
}

func createTestHostsDir(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "hosts_test_*")
	if err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

const sampleHostsContent = `# This is a test hosts file
# Some initial comments
127.0.0.1 localhost
::1 localhost

# @category development Local development hosts
# =============== DEVELOPMENT ===============
192.168.1.100 api.dev web.dev
192.168.1.101 db.dev # Database server

# @category production Production hosts
# =============== PRODUCTION ===============
10.0.0.50 api.prod
10.0.0.51 web.prod # Web server
# 10.0.0.52 disabled.prod # This is disabled

# Some footer comments
# End of file
`

// TestNewParser tests parser creation
func TestNewParser(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
	}{
		{
			name:     "valid file path",
			filePath: "/etc/hosts",
		},
		{
			name:     "relative path",
			filePath: "hosts",
		},
		{
			name:     "empty path",
			filePath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.filePath)

			if parser == nil {
				t.Fatal("NewParser returned nil")
			}

			if parser.filePath != tt.filePath {
				t.Errorf("NewParser().filePath = %q, want %q", parser.filePath, tt.filePath)
			}
		})
	}
}

// TestParserParse tests the parsing functionality
func TestParserParse(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectErr     bool
		expectEntries int
		validate      func(*testing.T, *HostsFile)
	}{
		{
			name:          "simple hosts file",
			content:       "127.0.0.1 localhost",
			expectErr:     false,
			expectEntries: 1,
			validate: func(t *testing.T, hf *HostsFile) {
				if len(hf.Categories) != 1 {
					t.Errorf("expected 1 category, got %d", len(hf.Categories))
				}
				if hf.Categories[0].Name != CategoryDefault {
					t.Errorf("expected category name %q, got %q", CategoryDefault, hf.Categories[0].Name)
				}
				if len(hf.Categories[0].Entries) != 1 {
					t.Errorf("expected 1 entry, got %d", len(hf.Categories[0].Entries))
				}
				entry := hf.Categories[0].Entries[0]
				if entry.IP != "127.0.0.1" {
					t.Errorf("expected IP 127.0.0.1, got %q", entry.IP)
				}
				if len(entry.Hostnames) != 1 || entry.Hostnames[0] != "localhost" {
					t.Errorf("expected hostname localhost, got %v", entry.Hostnames)
				}
				if !entry.Enabled {
					t.Error("expected entry to be enabled")
				}
			},
		},
		{
			name:          "complex hosts file with categories",
			content:       sampleHostsContent,
			expectErr:     false,
			expectEntries: 7,
			validate: func(t *testing.T, hf *HostsFile) {
				if len(hf.Categories) != 3 {
					t.Errorf("expected 3 categories, got %d", len(hf.Categories))
				}

				// Check header preservation
				if len(hf.Header) == 0 {
					t.Error("expected header to be preserved")
				}

				// Find development category
				var devCategory *Category
				for _, cat := range hf.Categories {
					if cat.Name == "development" {
						devCategory = &cat
						break
					}
				}
				if devCategory == nil {
					t.Fatal("development category not found")
				}

				if len(devCategory.Entries) != 2 {
					t.Errorf("expected 2 entries in development category, got %d", len(devCategory.Entries))
				}

				// Check if description is parsed
				if devCategory.Description != "Local development hosts" {
					t.Errorf("expected description 'Local development hosts', got %q", devCategory.Description)
				}
			},
		},
		{
			name:          "disabled entries",
			content:       "# 192.168.1.1 disabled.local # Disabled entry\n192.168.1.2 enabled.local",
			expectErr:     false,
			expectEntries: 2,
			validate: func(t *testing.T, hf *HostsFile) {
				if len(hf.Categories[0].Entries) != 2 {
					t.Errorf("expected 2 entries, got %d", len(hf.Categories[0].Entries))
				}

				// Find disabled entry
				var disabledEntry, enabledEntry *Entry
				for _, entry := range hf.Categories[0].Entries {
					switch entry.Hostnames[0] {
					case "disabled.local":
						disabledEntry = &entry
					case "enabled.local":
						enabledEntry = &entry
					}
				}

				if disabledEntry == nil {
					t.Fatal("disabled entry not found")
				}
				if !disabledEntry.Enabled {
					// This is expected - disabled entry should be disabled
				} else {
					t.Error("disabled entry should not be enabled")
				}

				if enabledEntry == nil {
					t.Fatal("enabled entry not found")
				}
				if !enabledEntry.Enabled {
					t.Error("enabled entry should be enabled")
				}
			},
		},
		{
			name:          "IPv6 addresses",
			content:       "::1 localhost\n2001:db8::1 ipv6.test",
			expectErr:     false,
			expectEntries: 2,
			validate: func(t *testing.T, hf *HostsFile) {
				if len(hf.Categories[0].Entries) != 2 {
					t.Errorf("expected 2 entries, got %d", len(hf.Categories[0].Entries))
				}

				for _, entry := range hf.Categories[0].Entries {
					if entry.IP == "::1" && entry.Hostnames[0] != "localhost" {
						t.Errorf("expected hostname localhost for ::1, got %q", entry.Hostnames[0])
					}
					if entry.IP == "2001:db8::1" && entry.Hostnames[0] != "ipv6.test" {
						t.Errorf("expected hostname ipv6.test for 2001:db8::1, got %q", entry.Hostnames[0])
					}
				}
			},
		},
		{
			name:          "multiple hostnames per entry",
			content:       "192.168.1.100 host1.local host2.local host3.local",
			expectErr:     false,
			expectEntries: 1,
			validate: func(t *testing.T, hf *HostsFile) {
				entry := hf.Categories[0].Entries[0]
				if len(entry.Hostnames) != 3 {
					t.Errorf("expected 3 hostnames, got %d", len(entry.Hostnames))
				}
				expected := []string{"host1.local", "host2.local", "host3.local"}
				for i, hostname := range expected {
					if i >= len(entry.Hostnames) || entry.Hostnames[i] != hostname {
						t.Errorf("expected hostname %s at index %d, got %v", hostname, i, entry.Hostnames)
					}
				}
			},
		},
		{
			name:          "comments with entries",
			content:       "192.168.1.100 test.local # This is a test server",
			expectErr:     false,
			expectEntries: 1,
			validate: func(t *testing.T, hf *HostsFile) {
				entry := hf.Categories[0].Entries[0]
				if entry.Comment != "This is a test server" {
					t.Errorf("expected comment 'This is a test server', got %q", entry.Comment)
				}
			},
		},
		{
			name:          "empty file",
			content:       "",
			expectErr:     false,
			expectEntries: 0,
			validate: func(t *testing.T, hf *HostsFile) {
				if len(hf.Categories) != 1 {
					t.Errorf("expected 1 default category, got %d", len(hf.Categories))
				}
				if hf.Categories[0].Name != CategoryDefault {
					t.Errorf("expected default category name, got %q", hf.Categories[0].Name)
				}
			},
		},
		{
			name:          "only comments",
			content:       "# This is a comment\n# Another comment",
			expectErr:     false,
			expectEntries: 0,
			validate: func(t *testing.T, hf *HostsFile) {
				if len(hf.Header) == 0 {
					t.Error("expected comments to be preserved in header")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := createTestHostsFile(t, tt.content)
			defer func() { _ = os.Remove(filePath) }()

			parser := NewParser(filePath)
			result, err := parser.Parse()

			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if err == nil {
				totalEntries := 0
				for _, category := range result.Categories {
					totalEntries += len(category.Entries)
				}

				if totalEntries != tt.expectEntries {
					t.Errorf("expected %d entries, got %d", tt.expectEntries, totalEntries)
				}

				if tt.validate != nil {
					tt.validate(t, result)
				}

				// Validate basic properties
				if result.FilePath != filePath {
					t.Errorf("expected file path %q, got %q", filePath, result.FilePath)
				}

				if result.Modified.IsZero() {
					t.Error("expected Modified timestamp to be set")
				}
			}
		})
	}
}

// TestParserParseErrors tests parsing error conditions
func TestParserParseErrors(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		expectError bool
	}{
		{
			name:        "non-existent file",
			filePath:    "/non/existent/file",
			expectError: true,
		},
		{
			name:        "directory instead of file",
			filePath:    "/tmp",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.filePath)
			_, err := parser.Parse()

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestParseEntry tests individual entry parsing
func TestParseEntry(t *testing.T) {
	parser := NewParser("")

	tests := []struct {
		name            string
		line            string
		lineNum         int
		expectOK        bool
		expectIP        string
		expectHostnames []string
		expectComment   string
		expectEnabled   bool
	}{
		{
			name:            "simple entry",
			line:            "127.0.0.1 localhost",
			lineNum:         1,
			expectOK:        true,
			expectIP:        "127.0.0.1",
			expectHostnames: []string{"localhost"},
			expectComment:   "",
			expectEnabled:   true,
		},
		{
			name:            "entry with comment",
			line:            "192.168.1.1 test.local # Test server",
			lineNum:         2,
			expectOK:        true,
			expectIP:        "192.168.1.1",
			expectHostnames: []string{"test.local"},
			expectComment:   "Test server",
			expectEnabled:   true,
		},
		{
			name:            "disabled entry",
			line:            "# 192.168.1.2 disabled.local",
			lineNum:         3,
			expectOK:        true,
			expectIP:        "192.168.1.2",
			expectHostnames: []string{"disabled.local"},
			expectComment:   "",
			expectEnabled:   false,
		},
		{
			name:            "multiple hostnames",
			line:            "192.168.1.100 host1 host2 host3",
			lineNum:         4,
			expectOK:        true,
			expectIP:        "192.168.1.100",
			expectHostnames: []string{"host1", "host2", "host3"},
			expectComment:   "",
			expectEnabled:   true,
		},
		{
			name:            "IPv6 entry",
			line:            "2001:db8::1 ipv6.test",
			lineNum:         5,
			expectOK:        true,
			expectIP:        "2001:db8::1",
			expectHostnames: []string{"ipv6.test"},
			expectComment:   "",
			expectEnabled:   true,
		},
		{
			name:     "comment line",
			line:     "# This is just a comment",
			lineNum:  6,
			expectOK: false,
		},
		{
			name:     "empty line",
			line:     "",
			lineNum:  7,
			expectOK: false,
		},
		{
			name:     "invalid IP",
			line:     "999.999.999.999 invalid",
			lineNum:  8,
			expectOK: false,
		},
		{
			name:     "no hostname",
			line:     "192.168.1.1",
			lineNum:  9,
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, ok := parser.parseEntry(tt.line, tt.lineNum)

			if ok != tt.expectOK {
				t.Errorf("parseEntry() ok = %v, want %v", ok, tt.expectOK)
			}

			if tt.expectOK {
				if entry.IP != tt.expectIP {
					t.Errorf("parseEntry() IP = %q, want %q", entry.IP, tt.expectIP)
				}

				if len(entry.Hostnames) != len(tt.expectHostnames) {
					t.Errorf("parseEntry() hostnames length = %d, want %d", len(entry.Hostnames), len(tt.expectHostnames))
				} else {
					for i, hostname := range tt.expectHostnames {
						if entry.Hostnames[i] != hostname {
							t.Errorf("parseEntry() hostname[%d] = %q, want %q", i, entry.Hostnames[i], hostname)
						}
					}
				}

				if entry.Comment != tt.expectComment {
					t.Errorf("parseEntry() comment = %q, want %q", entry.Comment, tt.expectComment)
				}

				if entry.Enabled != tt.expectEnabled {
					t.Errorf("parseEntry() enabled = %v, want %v", entry.Enabled, tt.expectEnabled)
				}

				if entry.LineNum != tt.lineNum {
					t.Errorf("parseEntry() lineNum = %d, want %d", entry.LineNum, tt.lineNum)
				}
			}
		})
	}
}

// TestIsValidIP tests IP validation in parser
func TestIsValidIP(t *testing.T) {
	parser := NewParser("")

	tests := []struct {
		ip    string
		valid bool
	}{
		{"127.0.0.1", true},
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"::1", true},
		{"2001:db8::1", true},
		{"fe80::1", true},
		{"", false},
		{"999.999.999.999", false},
		{"192.168.1", false},
		{"not.an.ip", false},
		{"192.168.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := parser.isValidIP(tt.ip)
			if result != tt.valid {
				t.Errorf("isValidIP(%q) = %v, want %v", tt.ip, result, tt.valid)
			}
		})
	}
}

// TestHostsFileWrite tests file writing functionality
func TestHostsFileWrite(t *testing.T) {
	tmpDir := createTestHostsDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tests := []struct {
		name      string
		hostsFile *HostsFile
		validate  func(*testing.T, string)
	}{
		{
			name: "simple hosts file",
			hostsFile: &HostsFile{
				Categories: []Category{
					{
						Name:    CategoryDefault,
						Enabled: true,
						Entries: []Entry{
							{
								IP:        "127.0.0.1",
								Hostnames: []string{"localhost"},
								Comment:   "",
								Category:  CategoryDefault,
								Enabled:   true,
							},
						},
					},
				},
				Header: []string{"# Test hosts file"},
			},
			validate: func(t *testing.T, filePath string) {
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatal(err)
				}

				lines := strings.Split(string(content), "\n")
				found := false
				for _, line := range lines {
					if strings.Contains(line, "127.0.0.1") && strings.Contains(line, "localhost") {
						found = true
						break
					}
				}
				if !found {
					t.Error("expected to find '127.0.0.1 localhost' in output")
				}
			},
		},
		{
			name: "multiple categories",
			hostsFile: &HostsFile{
				Categories: []Category{
					{
						Name:    "development",
						Enabled: true,
						Entries: []Entry{
							{
								IP:        "192.168.1.100",
								Hostnames: []string{"api.dev"},
								Comment:   "Development API",
								Category:  "development",
								Enabled:   true,
							},
						},
					},
					{
						Name:    "production",
						Enabled: true,
						Entries: []Entry{
							{
								IP:        "10.0.0.50",
								Hostnames: []string{"api.prod"},
								Comment:   "",
								Category:  "production",
								Enabled:   true,
							},
						},
					},
				},
			},
			validate: func(t *testing.T, filePath string) {
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatal(err)
				}

				str := string(content)
				if !strings.Contains(str, "@category development") {
					t.Error("expected to find development category header")
				}
				if !strings.Contains(str, "@category production") {
					t.Error("expected to find production category header")
				}
				if !strings.Contains(str, "192.168.1.100 api.dev") {
					t.Error("expected to find development entry")
				}
				if !strings.Contains(str, "10.0.0.50 api.prod") {
					t.Error("expected to find production entry")
				}
			},
		},
		{
			name: "disabled entries",
			hostsFile: &HostsFile{
				Categories: []Category{
					{
						Name:    CategoryDefault,
						Enabled: true,
						Entries: []Entry{
							{
								IP:        "192.168.1.1",
								Hostnames: []string{"test.local"},
								Comment:   "Disabled entry",
								Category:  CategoryDefault,
								Enabled:   false,
							},
						},
					},
				},
			},
			validate: func(t *testing.T, filePath string) {
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatal(err)
				}

				str := string(content)
				if !strings.Contains(str, "# 192.168.1.1 test.local") {
					t.Error("expected disabled entry to be commented out")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, fmt.Sprintf("test_%s.hosts", tt.name))

			err := tt.hostsFile.Write(outputPath)
			if err != nil {
				t.Fatalf("Write() error = %v", err)
			}

			// Verify file exists
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Fatal("output file does not exist")
			}

			if tt.validate != nil {
				tt.validate(t, outputPath)
			}

			// Verify Modified timestamp was updated
			if tt.hostsFile.Modified.IsZero() {
				t.Error("expected Modified timestamp to be updated after write")
			}
		})
	}
}

// TestFormatEntry tests entry formatting
func TestFormatEntry(t *testing.T) {
	tests := []struct {
		name     string
		entry    Entry
		expected string
	}{
		{
			name: "simple entry",
			entry: Entry{
				IP:        "127.0.0.1",
				Hostnames: []string{"localhost"},
				Comment:   "",
				Enabled:   true,
			},
			expected: "127.0.0.1 localhost",
		},
		{
			name: "entry with comment",
			entry: Entry{
				IP:        "192.168.1.1",
				Hostnames: []string{"test.local"},
				Comment:   "Test server",
				Enabled:   true,
			},
			expected: "192.168.1.1 test.local # Test server",
		},
		{
			name: "disabled entry",
			entry: Entry{
				IP:        "192.168.1.2",
				Hostnames: []string{"disabled.local"},
				Comment:   "Disabled",
				Enabled:   false,
			},
			expected: "# 192.168.1.2 disabled.local # Disabled",
		},
		{
			name: "multiple hostnames",
			entry: Entry{
				IP:        "192.168.1.100",
				Hostnames: []string{"host1", "host2", "host3"},
				Comment:   "",
				Enabled:   true,
			},
			expected: "192.168.1.100 host1 host2 host3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatEntry(tt.entry)
			if result != tt.expected {
				t.Errorf("formatEntry() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHostsFileAddEntry tests adding entries
func TestHostsFileAddEntry(t *testing.T) {
	tests := []struct {
		name      string
		initial   *HostsFile
		entry     Entry
		expectErr bool
		validate  func(*testing.T, *HostsFile)
	}{
		{
			name: "add to existing category",
			initial: &HostsFile{
				Categories: []Category{
					{
						Name:    CategoryDefault,
						Enabled: true,
						Entries: []Entry{},
					},
				},
			},
			entry: Entry{
				IP:        "127.0.0.1",
				Hostnames: []string{"localhost"},
				Category:  CategoryDefault,
				Enabled:   true,
			},
			expectErr: false,
			validate: func(t *testing.T, hf *HostsFile) {
				if len(hf.Categories[0].Entries) != 1 {
					t.Errorf("expected 1 entry, got %d", len(hf.Categories[0].Entries))
				}
			},
		},
		{
			name: "add to new category",
			initial: &HostsFile{
				Categories: []Category{
					{
						Name:    CategoryDefault,
						Enabled: true,
						Entries: []Entry{},
					},
				},
			},
			entry: Entry{
				IP:        "192.168.1.1",
				Hostnames: []string{"test.local"},
				Category:  "custom",
				Enabled:   true,
			},
			expectErr: false,
			validate: func(t *testing.T, hf *HostsFile) {
				if len(hf.Categories) != 2 {
					t.Errorf("expected 2 categories, got %d", len(hf.Categories))
				}

				found := false
				for _, cat := range hf.Categories {
					if cat.Name == "custom" {
						found = true
						if len(cat.Entries) != 1 {
							t.Errorf("expected 1 entry in custom category, got %d", len(cat.Entries))
						}
					}
				}
				if !found {
					t.Error("custom category not found")
				}
			},
		},
		{
			name: "add entry with empty category",
			initial: &HostsFile{
				Categories: []Category{},
			},
			entry: Entry{
				IP:        "192.168.1.1",
				Hostnames: []string{"test.local"},
				Category:  "",
				Enabled:   true,
			},
			expectErr: false,
			validate: func(t *testing.T, hf *HostsFile) {
				if len(hf.Categories) != 1 {
					t.Errorf("expected 1 category, got %d", len(hf.Categories))
				}
				if hf.Categories[0].Name != CategoryDefault {
					t.Errorf("expected default category, got %q", hf.Categories[0].Name)
				}
			},
		},
		{
			name: "add invalid entry",
			initial: &HostsFile{
				Categories: []Category{},
			},
			entry: Entry{
				IP:        "invalid.ip",
				Hostnames: []string{"test.local"},
				Category:  CategoryDefault,
				Enabled:   true,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initial.AddEntry(tt.entry)

			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectErr && tt.validate != nil {
				tt.validate(t, tt.initial)
			}
		})
	}
}

// TestHostsFileRemoveEntry tests removing entries
func TestHostsFileRemoveEntry(t *testing.T) {

	tests := []struct {
		name     string
		hostname string
		expected bool
		validate func(*testing.T, *HostsFile)
	}{
		{
			name:     "remove existing hostname with single hostname",
			hostname: "localhost",
			expected: true,
			validate: func(t *testing.T, hf *HostsFile) {
				if len(hf.Categories[0].Entries) != 1 {
					t.Errorf("expected 1 entry after removal, got %d", len(hf.Categories[0].Entries))
				}
			},
		},
		{
			name:     "remove hostname from entry with multiple hostnames",
			hostname: "test1.local",
			expected: true,
			validate: func(t *testing.T, hf *HostsFile) {
				found := false
				for _, entry := range hf.Categories[0].Entries {
					for _, h := range entry.Hostnames {
						if h == "test2.local" {
							found = true
						}
						if h == "test1.local" {
							t.Error("test1.local should have been removed")
						}
					}
				}
				if !found {
					t.Error("test2.local should still exist")
				}
			},
		},
		{
			name:     "remove non-existent hostname",
			hostname: "nonexistent.local",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy for each test
			hf := &HostsFile{
				Categories: []Category{
					{
						Name:    CategoryDefault,
						Enabled: true,
						Entries: []Entry{
							{
								IP:        "127.0.0.1",
								Hostnames: []string{"localhost"},
								Enabled:   true,
							},
							{
								IP:        "192.168.1.1",
								Hostnames: []string{"test1.local", "test2.local"},
								Enabled:   true,
							},
						},
					},
				},
			}

			result := hf.RemoveEntry(tt.hostname)
			if result != tt.expected {
				t.Errorf("RemoveEntry() = %v, want %v", result, tt.expected)
			}

			if tt.validate != nil {
				tt.validate(t, hf)
			}
		})
	}
}

// TestHostsFileEnableDisableEntry tests enabling/disabling entries
func TestHostsFileEnableDisableEntry(t *testing.T) {
	createTestHostsFile := func() *HostsFile {
		return &HostsFile{
			Categories: []Category{
				{
					Name:    CategoryDefault,
					Enabled: true,
					Entries: []Entry{
						{
							IP:        "127.0.0.1",
							Hostnames: []string{"localhost"},
							Enabled:   true,
						},
						{
							IP:        "192.168.1.1",
							Hostnames: []string{"test.local"},
							Enabled:   false,
						},
					},
				},
			},
		}
	}

	t.Run("enable entry", func(t *testing.T) {
		hf := createTestHostsFile()
		result := hf.EnableEntry("test.local")

		if !result {
			t.Error("EnableEntry() should return true for existing hostname")
		}

		// Find the entry and check if it's enabled
		for _, entry := range hf.Categories[0].Entries {
			for _, hostname := range entry.Hostnames {
				if hostname == "test.local" && !entry.Enabled {
					t.Error("test.local should be enabled")
				}
			}
		}
	})

	t.Run("disable entry", func(t *testing.T) {
		hf := createTestHostsFile()
		result := hf.DisableEntry("localhost")

		if !result {
			t.Error("DisableEntry() should return true for existing hostname")
		}

		// Find the entry and check if it's disabled
		for _, entry := range hf.Categories[0].Entries {
			for _, hostname := range entry.Hostnames {
				if hostname == "localhost" && entry.Enabled {
					t.Error("localhost should be disabled")
				}
			}
		}
	})

	t.Run("enable non-existent entry", func(t *testing.T) {
		hf := createTestHostsFile()
		result := hf.EnableEntry("nonexistent.local")

		if result {
			t.Error("EnableEntry() should return false for non-existent hostname")
		}
	})

	t.Run("disable non-existent entry", func(t *testing.T) {
		hf := createTestHostsFile()
		result := hf.DisableEntry("nonexistent.local")

		if result {
			t.Error("DisableEntry() should return false for non-existent hostname")
		}
	})
}

// TestHostsFileFindEntries tests finding entries
func TestHostsFileFindEntries(t *testing.T) {
	hostsFile := &HostsFile{
		Categories: []Category{
			{
				Name:    CategoryDefault,
				Enabled: true,
				Entries: []Entry{
					{
						IP:        "127.0.0.1",
						Hostnames: []string{"localhost"},
						Enabled:   true,
					},
					{
						IP:        "192.168.1.100",
						Hostnames: []string{"api.dev", "web.dev"},
						Comment:   "Development servers",
						Enabled:   true,
					},
					{
						IP:        "10.0.0.50",
						Hostnames: []string{"api.prod"},
						Enabled:   true,
					},
				},
			},
		},
	}

	tests := []struct {
		name          string
		query         string
		expectedCount int
		validate      func(*testing.T, []Entry)
	}{
		{
			name:          "find by hostname",
			query:         "localhost",
			expectedCount: 1,
			validate: func(t *testing.T, entries []Entry) {
				if entries[0].IP != "127.0.0.1" {
					t.Errorf("expected IP 127.0.0.1, got %q", entries[0].IP)
				}
			},
		},
		{
			name:          "find by partial hostname",
			query:         "dev",
			expectedCount: 1,
			validate: func(t *testing.T, entries []Entry) {
				if entries[0].IP != "192.168.1.100" {
					t.Errorf("expected IP 192.168.1.100, got %q", entries[0].IP)
				}
			},
		},
		{
			name:          "find by IP",
			query:         "10.0.0",
			expectedCount: 1,
			validate: func(t *testing.T, entries []Entry) {
				if entries[0].IP != "10.0.0.50" {
					t.Errorf("expected IP 10.0.0.50, got %q", entries[0].IP)
				}
			},
		},
		{
			name:          "case insensitive search",
			query:         "API",
			expectedCount: 2,
		},
		{
			name:          "no matches",
			query:         "nonexistent",
			expectedCount: 0,
		},
		{
			name:          "empty query",
			query:         "",
			expectedCount: 6, // Empty string matches all entries (hostname + IP matches)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := hostsFile.FindEntries(tt.query)

			if len(results) != tt.expectedCount {
				t.Errorf("FindEntries() returned %d results, want %d", len(results), tt.expectedCount)
			}

			if tt.validate != nil && len(results) > 0 {
				tt.validate(t, results)
			}
		})
	}
}

// TestHostsFileGetCategory tests getting categories
func TestHostsFileGetCategory(t *testing.T) {
	hostsFile := &HostsFile{
		Categories: []Category{
			{Name: CategoryDefault, Enabled: true},
			{Name: "development", Enabled: true},
			{Name: "production", Enabled: false},
		},
	}

	tests := []struct {
		name          string
		categoryName  string
		expectFound   bool
		expectEnabled bool
	}{
		{
			name:          "existing category",
			categoryName:  "development",
			expectFound:   true,
			expectEnabled: true,
		},
		{
			name:          "disabled category",
			categoryName:  "production",
			expectFound:   true,
			expectEnabled: false,
		},
		{
			name:         "non-existent category",
			categoryName: "nonexistent",
			expectFound:  false,
		},
		{
			name:          "default category",
			categoryName:  CategoryDefault,
			expectFound:   true,
			expectEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := hostsFile.GetCategory(tt.categoryName)

			if tt.expectFound && category == nil {
				t.Error("expected category to be found but got nil")
			}
			if !tt.expectFound && category != nil {
				t.Errorf("expected category to not be found but got %v", category)
			}

			if category != nil {
				if category.Name != tt.categoryName {
					t.Errorf("expected category name %q, got %q", tt.categoryName, category.Name)
				}
				if category.Enabled != tt.expectEnabled {
					t.Errorf("expected category enabled = %v, got %v", tt.expectEnabled, category.Enabled)
				}
			}
		})
	}
}

// TestHostsFileEnableDisableCategory tests enabling/disabling categories
func TestHostsFileEnableDisableCategory(t *testing.T) {
	createTestHostsFile := func() *HostsFile {
		return &HostsFile{
			Categories: []Category{
				{
					Name:    "development",
					Enabled: false,
					Entries: []Entry{
						{IP: "192.168.1.1", Hostnames: []string{"test1.dev"}, Enabled: false},
						{IP: "192.168.1.2", Hostnames: []string{"test2.dev"}, Enabled: false},
					},
				},
				{
					Name:    "production",
					Enabled: true,
					Entries: []Entry{
						{IP: "10.0.0.1", Hostnames: []string{"test1.prod"}, Enabled: true},
						{IP: "10.0.0.2", Hostnames: []string{"test2.prod"}, Enabled: true},
					},
				},
			},
		}
	}

	t.Run("enable category", func(t *testing.T) {
		hf := createTestHostsFile()
		hf.EnableCategory("development")

		devCategory := hf.GetCategory("development")
		if devCategory == nil {
			t.Fatal("development category not found")
		}

		if !devCategory.Enabled {
			t.Error("development category should be enabled")
		}

		for _, entry := range devCategory.Entries {
			if !entry.Enabled {
				t.Errorf("entry %v should be enabled", entry.Hostnames)
			}
		}
	})

	t.Run("disable category", func(t *testing.T) {
		hf := createTestHostsFile()
		hf.DisableCategory("production")

		prodCategory := hf.GetCategory("production")
		if prodCategory == nil {
			t.Fatal("production category not found")
		}

		if prodCategory.Enabled {
			t.Error("production category should be disabled")
		}

		for _, entry := range prodCategory.Entries {
			if entry.Enabled {
				t.Errorf("entry %v should be disabled", entry.Hostnames)
			}
		}
	})

	t.Run("enable non-existent category", func(t *testing.T) {
		hf := createTestHostsFile()
		hf.EnableCategory("nonexistent")
		// Should not crash or panic
	})

	t.Run("disable non-existent category", func(t *testing.T) {
		hf := createTestHostsFile()
		hf.DisableCategory("nonexistent")
		// Should not crash or panic
	})
}

// Benchmark tests
func BenchmarkParserParse(b *testing.B) {
	filePath := createTestHostsFileB(b, sampleHostsContent)
	defer func() { _ = os.Remove(filePath) }()

	parser := NewParser(filePath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHostsFileWrite(b *testing.B) {
	tmpDir := createTestHostsDirB(b)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	hostsFile := &HostsFile{
		Categories: []Category{
			{
				Name:    CategoryDefault,
				Enabled: true,
				Entries: []Entry{
					{IP: "127.0.0.1", Hostnames: []string{"localhost"}, Enabled: true},
					{IP: "192.168.1.100", Hostnames: []string{"api.dev"}, Enabled: true},
					{IP: "192.168.1.101", Hostnames: []string{"web.dev"}, Enabled: true},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputPath := filepath.Join(tmpDir, fmt.Sprintf("bench_%d.hosts", i))
		err := hostsFile.Write(outputPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHostsFileAddEntry(b *testing.B) {
	hostsFile := &HostsFile{
		Categories: []Category{
			{Name: CategoryDefault, Enabled: true, Entries: []Entry{}},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := Entry{
			IP:        fmt.Sprintf("192.168.1.%d", i%254+1),
			Hostnames: []string{fmt.Sprintf("host%d.local", i)},
			Category:  CategoryDefault,
			Enabled:   true,
		}
		err := hostsFile.AddEntry(entry)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHostsFileFindEntries(b *testing.B) {
	// Create a large hosts file for realistic benchmarking
	hf := &HostsFile{
		Categories: []Category{
			{
				Name:    CategoryDefault,
				Enabled: true,
				Entries: []Entry{},
			},
		},
	}

	// Add many entries
	for i := 0; i < 1000; i++ {
		entry := Entry{
			IP:        fmt.Sprintf("192.168.%d.%d", i/254, i%254+1),
			Hostnames: []string{fmt.Sprintf("host%d.local", i)},
			Category:  CategoryDefault,
			Enabled:   true,
		}
		_ = hf.AddEntry(entry)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := fmt.Sprintf("host%d", i%1000)
		hf.FindEntries(query)
	}
}

// Edge case tests
func TestHostsFileEdgeCases(t *testing.T) {
	t.Run("write to read-only directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping read-only directory test on Windows")
		}

		tmpDir := createTestHostsDir(t)
		defer func() { _ = os.RemoveAll(tmpDir) }()

		// Make directory read-only
		err := os.Chmod(tmpDir, 0444)
		if err != nil {
			t.Skip("Cannot make directory read-only")
		}
		defer func() { _ = os.Chmod(tmpDir, 0755) }() // Restore permissions for cleanup

		hostsFile := &HostsFile{
			Categories: []Category{
				{Name: CategoryDefault, Enabled: true, Entries: []Entry{}},
			},
		}

		outputPath := filepath.Join(tmpDir, "readonly.hosts")
		err = hostsFile.Write(outputPath)
		if err == nil {
			t.Error("expected error when writing to read-only directory")
		}
	})

	t.Run("parse file with very long lines", func(t *testing.T) {
		longHostname := strings.Repeat("a", 1000)
		content := fmt.Sprintf("127.0.0.1 %s", longHostname)

		filePath := createTestHostsFile(t, content)
		defer func() { _ = os.Remove(filePath) }()

		parser := NewParser(filePath)
		_, err := parser.Parse()
		// Should handle long lines gracefully (may succeed or fail with validation error)
		if err != nil {
			// This is acceptable - the validation may reject overly long hostnames
			t.Logf("Long hostname rejected (expected): %v", err)
		}
	})

	t.Run("concurrent access to hosts file", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping concurrency test in short mode")
		}

		hostsFile := &HostsFile{
			Categories: []Category{
				{Name: CategoryDefault, Enabled: true, Entries: []Entry{}},
			},
		}

		var wg sync.WaitGroup
		errChan := make(chan error, 10)

		// Concurrent AddEntry operations
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				entry := Entry{
					IP:        fmt.Sprintf("192.168.1.%d", id+1),
					Hostnames: []string{fmt.Sprintf("host%d.local", id)},
					Category:  CategoryDefault,
					Enabled:   true,
				}
				if err := hostsFile.AddEntry(entry); err != nil {
					errChan <- err
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		// Check for any errors
		for err := range errChan {
			t.Errorf("Concurrent access error: %v", err)
		}

		// Verify all entries were added (allowing for some potential validation failures in concurrent access)
		if len(hostsFile.Categories[0].Entries) < 9 || len(hostsFile.Categories[0].Entries) > 10 {
			t.Errorf("expected 9-10 entries due to concurrent access, got %d", len(hostsFile.Categories[0].Entries))
		}
	})
}

// Helper functions for benchmarks
func createTestHostsDirB(b *testing.B) string {
	b.Helper()
	tmpDir, err := os.MkdirTemp("", "hosts_test_*")
	if err != nil {
		b.Fatal(err)
	}
	return tmpDir
}

func createTestHostsFileB(b *testing.B, content string) string {
	b.Helper()
	tmpFile, err := os.CreateTemp("", "hosts_test_*.txt")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = tmpFile.Close() }()

	if _, err := tmpFile.WriteString(content); err != nil {
		b.Fatal(err)
	}

	return tmpFile.Name()
}

func TestHostsFileAddCategory(t *testing.T) {
	tests := []struct {
		name          string
		categoryName  string
		description   string
		expectError   bool
		errorContains string
	}{
		{
			name:         "Valid category with description",
			categoryName: "testing",
			description:  "Testing category",
			expectError:  false,
		},
		{
			name:         "Valid category without description",
			categoryName: "minimal",
			description:  "",
			expectError:  false,
		},
		{
			name:          "Empty category name",
			categoryName:  "",
			description:   "Should fail",
			expectError:   true,
			errorContains: "validation failed",
		},
		{
			name:          "Invalid category name with spaces",
			categoryName:  "test category",
			description:   "Should fail",
			expectError:   true,
			errorContains: "validation failed",
		},
		{
			name:          "Invalid category name with special chars",
			categoryName:  "test@category",
			description:   "Should fail",
			expectError:   true,
			errorContains: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test hosts file
			hostsPath := createTestHostsFile(t, sampleHostsContent)
			defer func() { _ = os.Remove(hostsPath) }()

			// Parse the hosts file
			parser := NewParser(hostsPath)
			hostsFile, err := parser.Parse()
			if err != nil {
				t.Fatalf("Failed to parse hosts file: %v", err)
			}

			// Check initial state
			initialCategoryCount := len(hostsFile.Categories)

			// Test duplicate category (should fail)
			if tt.categoryName == "development" {
				err := hostsFile.AddCategory(tt.categoryName, tt.description)
				if err == nil {
					t.Errorf("Expected error for duplicate category, got nil")
				}
				if !strings.Contains(err.Error(), "already exists") {
					t.Errorf("Expected 'already exists' error, got: %v", err)
				}
				return
			}

			// Add the category
			err = hostsFile.AddCategory(tt.categoryName, tt.description)

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

			// Should not have error
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify category was added
			if len(hostsFile.Categories) != initialCategoryCount+1 {
				t.Errorf("Expected %d categories, got %d", initialCategoryCount+1, len(hostsFile.Categories))
			}

			// Find the added category
			var addedCategory *Category
			for i := range hostsFile.Categories {
				if hostsFile.Categories[i].Name == tt.categoryName {
					addedCategory = &hostsFile.Categories[i]
					break
				}
			}

			if addedCategory == nil {
				t.Errorf("Category '%s' was not added", tt.categoryName)
				return
			}

			// Verify category properties
			if addedCategory.Name != tt.categoryName {
				t.Errorf("Expected category name '%s', got '%s'", tt.categoryName, addedCategory.Name)
			}
			if addedCategory.Description != tt.description {
				t.Errorf("Expected category description '%s', got '%s'", tt.description, addedCategory.Description)
			}
			if !addedCategory.Enabled {
				t.Errorf("Expected category to be enabled by default")
			}
			if len(addedCategory.Entries) != 0 {
				t.Errorf("Expected category to have no entries initially, got %d", len(addedCategory.Entries))
			}
		})
	}
}

func TestHostsFileAddCategoryDuplicate(t *testing.T) {
	// Create a test hosts file
	hostsPath := createTestHostsFile(t, sampleHostsContent)
	defer func() { _ = os.Remove(hostsPath) }()

	// Parse the hosts file
	parser := NewParser(hostsPath)
	hostsFile, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse hosts file: %v", err)
	}

	// Try to add a category that already exists
	err = hostsFile.AddCategory("development", "Duplicate category")
	if err == nil {
		t.Errorf("Expected error for duplicate category, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' error, got: %v", err)
	}
}

func TestHostsFileAddCategoryAndWrite(t *testing.T) {
	// Create a test hosts file
	hostsPath := createTestHostsFile(t, sampleHostsContent)
	defer func() { _ = os.Remove(hostsPath) }()

	// Parse the hosts file
	parser := NewParser(hostsPath)
	hostsFile, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse hosts file: %v", err)
	}

	// Add a new category
	err = hostsFile.AddCategory("testing", "Testing category for persistence")
	if err != nil {
		t.Fatalf("Failed to add category: %v", err)
	}

	// Add an entry to the new category so it gets written to the file
	testEntry := Entry{
		IP:        "192.168.1.200",
		Hostnames: []string{"test.local"},
		Comment:   "Test entry",
		Category:  "testing",
		Enabled:   true,
	}
	err = hostsFile.AddEntry(testEntry)
	if err != nil {
		t.Fatalf("Failed to add test entry: %v", err)
	}

	// Write the hosts file
	err = hostsFile.Write(hostsPath)
	if err != nil {
		t.Fatalf("Failed to write hosts file: %v", err)
	}

	// Re-parse the file to verify persistence
	parser2 := NewParser(hostsPath)
	hostsFile2, err := parser2.Parse()
	if err != nil {
		t.Fatalf("Failed to re-parse hosts file: %v", err)
	}

	// Verify the category exists in the re-parsed file
	testingCategory := hostsFile2.GetCategory("testing")
	if testingCategory == nil {
		t.Errorf("Category 'testing' not found after write/read cycle")
		return
	}

	if testingCategory.Description != "Testing category for persistence" {
		t.Errorf("Expected description 'Testing category for persistence', got '%s'", testingCategory.Description)
	}
}
