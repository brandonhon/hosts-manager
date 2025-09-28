package config

import (
	"strings"
	"testing"
)

func TestValidationError(t *testing.T) {
	err := ValidationError{
		Field:   "test.field",
		Value:   "test_value",
		Message: "test message",
	}

	expected := "validation error in field 'test.field': test message (value: test_value)"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestNewValidator(t *testing.T) {
	validator := NewValidator()
	if validator == nil {
		t.Fatal("Expected validator to be created")
	}
	if validator.errors == nil {
		t.Error("Expected errors slice to be initialized")
	}
	if len(validator.errors) != 0 {
		t.Error("Expected empty errors slice")
	}
}

func TestValidateValidConfig(t *testing.T) {
	config := DefaultConfig()
	validator := NewValidator()

	err := validator.Validate(config)
	if err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}
}

func TestValidateGeneral(t *testing.T) {
	tests := []struct {
		name          string
		general       General
		expectError   bool
		errorContains string
	}{
		{
			name: "valid general config",
			general: General{
				DefaultCategory: "custom",
				AutoBackup:      true,
				DryRun:          false,
				Verbose:         false,
				Editor:          "nano",
			},
			expectError: false,
		},
		{
			name: "empty default category",
			general: General{
				DefaultCategory: "",
				Editor:          "nano",
			},
			expectError:   true,
			errorContains: "default category cannot be empty",
		},
		{
			name: "invalid category name",
			general: General{
				DefaultCategory: "invalid@category",
				Editor:          "nano",
			},
			expectError:   true,
			errorContains: "invalid category name format",
		},
		{
			name: "invalid editor",
			general: General{
				DefaultCategory: "custom",
				Editor:          "malicious_command; rm -rf /",
			},
			expectError:   true,
			errorContains: "invalid or potentially unsafe editor",
		},
		{
			name: "valid editor with arguments",
			general: General{
				DefaultCategory: "custom",
				Editor:          "code --wait",
			},
			expectError: false, // code with safe arguments should be valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.General = tt.general
			validator := NewValidator()
			err := validator.Validate(config)

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectError && err != nil && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			}
		})
	}
}

func TestValidateCategories(t *testing.T) {
	tests := []struct {
		name          string
		categories    map[string]string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid categories",
			categories: map[string]string{
				"development": "Development environment",
				"production":  "Production environment",
			},
			expectError: false,
		},
		{
			name:          "empty categories",
			categories:    map[string]string{},
			expectError:   true,
			errorContains: "at least one category must be defined",
		},
		{
			name: "invalid category name",
			categories: map[string]string{
				"invalid@name": "Description",
			},
			expectError:   true,
			errorContains: "invalid category name format",
		},
		{
			name: "description too long",
			categories: map[string]string{
				"test": "This is a very long description that exceeds the maximum allowed length of 200 characters. This description is intentionally made very long to test the validation logic that should reject descriptions that are too long for the configuration system to handle properly and safely.",
			},
			expectError:   true,
			errorContains: "description too long",
		},
		{
			name: "suspicious content in description",
			categories: map[string]string{
				"test": "Description with <script>alert('xss')</script>",
			},
			expectError:   true,
			errorContains: "potentially dangerous content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Categories = tt.categories
			validator := NewValidator()
			err := validator.Validate(config)

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateProfiles(t *testing.T) {
	tests := []struct {
		name          string
		profiles      map[string]Profile
		expectError   bool
		errorContains string
	}{
		{
			name: "valid profiles",
			profiles: map[string]Profile{
				"development": {
					Description: "Development profile",
					Categories:  []string{"development", "staging"},
					Default:     false,
				},
				"production": {
					Description: "Production profile",
					Categories:  []string{"production"},
					Default:     true,
				},
			},
			expectError: false,
		},
		{
			name: "invalid profile name",
			profiles: map[string]Profile{
				"invalid@name": {
					Description: "Test profile",
					Categories:  []string{"development"},
					Default:     true,
				},
			},
			expectError:   true,
			errorContains: "invalid profile name format",
		},
		{
			name: "description too long",
			profiles: map[string]Profile{
				"test": {
					Description: "This is a very long description that exceeds the maximum allowed length of 200 characters. This description is intentionally made very long to test the validation logic that should reject descriptions that are too long for the configuration system to handle properly and safely.",
					Categories:  []string{"development"},
					Default:     true,
				},
			},
			expectError:   true,
			errorContains: "description too long",
		},
		{
			name: "suspicious content in description",
			profiles: map[string]Profile{
				"test": {
					Description: "Profile with javascript:alert('xss')",
					Categories:  []string{"development"},
					Default:     true,
				},
			},
			expectError:   true,
			errorContains: "potentially dangerous content",
		},
		{
			name: "empty categories",
			profiles: map[string]Profile{
				"test": {
					Description: "Test profile",
					Categories:  []string{},
					Default:     true,
				},
			},
			expectError:   true,
			errorContains: "must have at least one category",
		},
		{
			name: "invalid category in profile",
			profiles: map[string]Profile{
				"test": {
					Description: "Test profile",
					Categories:  []string{"invalid@category"},
					Default:     true,
				},
			},
			expectError:   true,
			errorContains: "invalid category name in profile",
		},
		{
			name: "no default profile",
			profiles: map[string]Profile{
				"test1": {
					Description: "Test profile 1",
					Categories:  []string{"development"},
					Default:     false,
				},
				"test2": {
					Description: "Test profile 2",
					Categories:  []string{"production"},
					Default:     false,
				},
			},
			expectError:   true,
			errorContains: "exactly one profile must be marked as default",
		},
		{
			name: "multiple default profiles",
			profiles: map[string]Profile{
				"test1": {
					Description: "Test profile 1",
					Categories:  []string{"development"},
					Default:     true,
				},
				"test2": {
					Description: "Test profile 2",
					Categories:  []string{"production"},
					Default:     true,
				},
			},
			expectError:   true,
			errorContains: "only one profile can be marked as default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Profiles = tt.profiles
			validator := NewValidator()
			err := validator.Validate(config)

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateUI(t *testing.T) {
	tests := []struct {
		name          string
		ui            UI
		expectError   bool
		errorContains string
	}{
		{
			name: "valid UI config",
			ui: UI{
				ColorScheme:     "auto",
				ShowLineNumbers: true,
				PageSize:        20,
				KeyBindings: map[string]string{
					"quit": "q",
					"help": "?",
				},
			},
			expectError: false,
		},
		{
			name: "invalid color scheme",
			ui: UI{
				ColorScheme: "invalid_scheme",
				PageSize:    20,
			},
			expectError:   true,
			errorContains: "invalid color scheme",
		},
		{
			name: "page size too small",
			ui: UI{
				ColorScheme: "auto",
				PageSize:    0,
			},
			expectError:   true,
			errorContains: "page size must be between 1 and 1000",
		},
		{
			name: "page size too large",
			ui: UI{
				ColorScheme: "auto",
				PageSize:    1001,
			},
			expectError:   true,
			errorContains: "page size must be between 1 and 1000",
		},
		{
			name: "invalid key binding",
			ui: UI{
				ColorScheme: "auto",
				PageSize:    20,
				KeyBindings: map[string]string{
					"quit": "invalid_key_binding_that_is_too_long",
				},
			},
			expectError:   true,
			errorContains: "invalid key binding format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.UI = tt.ui
			validator := NewValidator()
			err := validator.Validate(config)

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateBackup(t *testing.T) {
	tests := []struct {
		name          string
		backup        Backup
		expectError   bool
		errorContains string
	}{
		{
			name: "valid backup config",
			backup: Backup{
				Directory:       "/safe/path/backups",
				MaxBackups:      10,
				RetentionDays:   30,
				CompressionType: "gzip",
			},
			expectError: false,
		},
		{
			name: "suspicious directory path",
			backup: Backup{
				Directory:       "../../../etc/passwd",
				MaxBackups:      10,
				RetentionDays:   30,
				CompressionType: "gzip",
			},
			expectError:   true,
			errorContains: "potentially unsafe directory path",
		},
		{
			name: "max backups too small",
			backup: Backup{
				Directory:       "/safe/path",
				MaxBackups:      0,
				RetentionDays:   30,
				CompressionType: "gzip",
			},
			expectError:   true,
			errorContains: "max backups must be between 1 and 100",
		},
		{
			name: "max backups too large",
			backup: Backup{
				Directory:       "/safe/path",
				MaxBackups:      101,
				RetentionDays:   30,
				CompressionType: "gzip",
			},
			expectError:   true,
			errorContains: "max backups must be between 1 and 100",
		},
		{
			name: "retention days too small",
			backup: Backup{
				Directory:       "/safe/path",
				MaxBackups:      10,
				RetentionDays:   0,
				CompressionType: "gzip",
			},
			expectError:   true,
			errorContains: "retention days must be between 1 and 3650",
		},
		{
			name: "retention days too large",
			backup: Backup{
				Directory:       "/safe/path",
				MaxBackups:      10,
				RetentionDays:   3651,
				CompressionType: "gzip",
			},
			expectError:   true,
			errorContains: "retention days must be between 1 and 3650",
		},
		{
			name: "invalid compression type",
			backup: Backup{
				Directory:       "/safe/path",
				MaxBackups:      10,
				RetentionDays:   30,
				CompressionType: "invalid",
			},
			expectError:   true,
			errorContains: "invalid compression type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Backup = tt.backup
			validator := NewValidator()
			err := validator.Validate(config)

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateExport(t *testing.T) {
	tests := []struct {
		name          string
		export        Export
		expectError   bool
		errorContains string
	}{
		{
			name: "valid export config",
			export: Export{
				DefaultFormat: "yaml",
				Formats: map[string]Format{
					"yaml": {
						Extension: ".yaml",
						Template:  "{{.}}",
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty default format",
			export: Export{
				DefaultFormat: "",
				Formats: map[string]Format{
					"yaml": {
						Extension: ".yaml",
						Template:  "{{.}}",
					},
				},
			},
			expectError:   true,
			errorContains: "default format cannot be empty",
		},
		{
			name: "invalid format name",
			export: Export{
				DefaultFormat: "yaml",
				Formats: map[string]Format{
					"invalid@format": {
						Extension: ".yaml",
						Template:  "{{.}}",
					},
				},
			},
			expectError:   true,
			errorContains: "invalid format name",
		},
		{
			name: "invalid extension",
			export: Export{
				DefaultFormat: "yaml",
				Formats: map[string]Format{
					"yaml": {
						Extension: "yaml", // Missing dot
						Template:  "{{.}}",
					},
				},
			},
			expectError:   true,
			errorContains: "extension must start with '.'",
		},
		{
			name: "suspicious template content",
			export: Export{
				DefaultFormat: "yaml",
				Formats: map[string]Format{
					"yaml": {
						Extension: ".yaml",
						Template:  "{{exec \"rm -rf /\"}}",
					},
				},
			},
			expectError:   true,
			errorContains: "potentially dangerous content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Export = tt.export
			validator := NewValidator()
			err := validator.Validate(config)

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test isValidCategoryName
	validCategoryNames := []string{"development", "test_category", "prod-env", "cat1"}
	for _, name := range validCategoryNames {
		if !isValidCategoryName(name) {
			t.Errorf("Expected '%s' to be valid category name", name)
		}
	}

	invalidCategoryNames := []string{"", "invalid@name", "name with spaces", "very_long_category_name_that_exceeds_the_fifty_character_limit"}
	for _, name := range invalidCategoryNames {
		if isValidCategoryName(name) {
			t.Errorf("Expected '%s' to be invalid category name", name)
		}
	}

	// Test isValidProfileName
	validProfileNames := []string{"development", "test_profile", "prod-env", "prof1"}
	for _, name := range validProfileNames {
		if !isValidProfileName(name) {
			t.Errorf("Expected '%s' to be valid profile name", name)
		}
	}

	invalidProfileNames := []string{"", "invalid@profile", "profile with spaces", "very_long_profile_name_that_exceeds_the_fifty_character_limit"}
	for _, name := range invalidProfileNames {
		if isValidProfileName(name) {
			t.Errorf("Expected '%s' to be invalid profile name", name)
		}
	}

	// Test isValidFormatName
	validFormatNames := []string{"yaml", "json", "csv", "format-1"}
	for _, name := range validFormatNames {
		if !isValidFormatName(name) {
			t.Errorf("Expected '%s' to be valid format name", name)
		}
	}

	invalidFormatNames := []string{"", "format@name", "very_long_format_name_that_exceeds_limit"}
	for _, name := range invalidFormatNames {
		if isValidFormatName(name) {
			t.Errorf("Expected '%s' to be invalid format name", name)
		}
	}

	// Test isValidEditor
	validEditors := []string{"nano", "vim", "vi", "emacs", "code", "notepad", "notepad++", "sublime_text", "atom", "gedit", "kate"}
	for _, editor := range validEditors {
		if !isValidEditor(editor) {
			t.Errorf("Expected '%s' to be valid editor", editor)
		}
	}

	// Test editors with .exe extension (Windows)
	windowsEditors := []string{"notepad.exe", "code.exe", "vim.exe"}
	for _, editor := range windowsEditors {
		if !isValidEditor(editor) {
			t.Errorf("Expected '%s' to be valid Windows editor", editor)
		}
	}

	invalidEditors := []string{
		"rm -rf /",
		"editor; rm file",
		"editor && malicious",
		"editor | dangerous",
		"editor`malicious`",
		"editor$PWD",
		"editor\nmalicious",
		"editor\rmalicious",
		"",    // empty string
		"   ", // whitespace only
		"nonexistent_editor",
		"invalid@editor",
	}
	for _, editor := range invalidEditors {
		if isValidEditor(editor) {
			t.Errorf("Expected '%s' to be invalid editor", editor)
		}
	}

	// Test isValidKeyBinding
	validKeyBindings := []string{"q", "?", "space", "enter", "ctrl+c", "F1", "F12"}
	for _, key := range validKeyBindings {
		if !isValidKeyBinding(key) {
			t.Errorf("Expected '%s' to be valid key binding", key)
		}
	}

	invalidKeyBindings := []string{"", "very_long_key", "invalid@key"}
	for _, key := range invalidKeyBindings {
		if isValidKeyBinding(key) {
			t.Errorf("Expected '%s' to be invalid key binding", key)
		}
	}

	// Test containsSuspiciousContent
	suspiciousContent := []string{
		"<script>alert('xss')</script>",
		"javascript:alert('xss')",
		"onclick=\"malicious()\"",
		"eval(dangerous_code)",
	}
	for _, content := range suspiciousContent {
		if !containsSuspiciousContent(content) {
			t.Errorf("Expected '%s' to be detected as suspicious", content)
		}
	}

	safeContent := []string{
		"Normal description",
		"Development environment",
		"Production ready",
	}
	for _, content := range safeContent {
		if containsSuspiciousContent(content) {
			t.Errorf("Expected '%s' to be safe content", content)
		}
	}

	// Test containsSuspiciousPath
	suspiciousPaths := []string{
		"../../../etc/passwd",
		"/etc/shadow",
		"C:\\Windows\\System32",
		"path/with/\x00/null",
	}
	for _, path := range suspiciousPaths {
		if !containsSuspiciousPath(path) {
			t.Errorf("Expected '%s' to be detected as suspicious path", path)
		}
	}

	safePaths := []string{
		"/home/user/backups",
		"C:\\Users\\User\\AppData\\backups",
		"backups",
		"./local/backups",
	}
	for _, path := range safePaths {
		if containsSuspiciousPath(path) {
			t.Errorf("Expected '%s' to be safe path", path)
		}
	}

	// Test containsSuspiciousTemplate
	suspiciousTemplates := []string{
		"{{exec \"rm -rf /\"}}",
		"{{call os.Remove \"/important/file\"}}",
		"{{with runtime.GOMAXPROCS}}",
		"import \"os\"",
	}
	for _, template := range suspiciousTemplates {
		if !containsSuspiciousTemplate(template) {
			t.Errorf("Expected '%s' to be detected as suspicious template", template)
		}
	}

	safeTemplates := []string{
		"{{.Name}}",
		"{{range .Items}}{{.Value}}{{end}}",
		"# Generated by hosts-manager\\n{{.Content}}",
	}
	for _, template := range safeTemplates {
		if containsSuspiciousTemplate(template) {
			t.Errorf("Expected '%s' to be safe template", template)
		}
	}
}

func TestMultipleValidationErrors(t *testing.T) {
	config := DefaultConfig()

	// Create config with multiple validation errors
	config.General.DefaultCategory = ""      // Error 1: empty category
	config.General.Editor = "evil; rm -rf /" // Error 2: unsafe editor
	config.UI.PageSize = 0                   // Error 3: invalid page size
	config.Backup.MaxBackups = 0             // Error 4: invalid max backups

	validator := NewValidator()
	err := validator.Validate(config)

	if err == nil {
		t.Fatal("Expected validation errors but got none")
	}

	// Check that error mentions multiple validation errors
	errorStr := err.Error()
	if !strings.Contains(errorStr, "4 errors") {
		t.Errorf("Expected error to mention 4 validation errors, got: %s", errorStr)
	}
}

// BenchmarkValidateConfig benchmarks config validation
func BenchmarkValidateConfig(b *testing.B) {
	config := DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator := NewValidator()
		err := validator.Validate(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidateGeneral benchmarks general section validation
func BenchmarkValidateGeneral(b *testing.B) {
	general := DefaultConfig().General

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator := NewValidator()
		validator.validateGeneral(&general)
	}
}

// BenchmarkIsValidEditor benchmarks editor validation
func BenchmarkIsValidEditor(b *testing.B) {
	editors := []string{"nano", "vim", "code", "malicious; rm -rf /"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, editor := range editors {
			isValidEditor(editor)
		}
	}
}
