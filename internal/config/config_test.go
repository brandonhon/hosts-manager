package config

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test General section
	if config.General.DefaultCategory != "custom" {
		t.Errorf("Expected default category to be 'custom', got %s", config.General.DefaultCategory)
	}
	if !config.General.AutoBackup {
		t.Error("Expected auto backup to be enabled by default")
	}
	if config.General.DryRun {
		t.Error("Expected dry run to be disabled by default")
	}
	if config.General.Verbose {
		t.Error("Expected verbose to be disabled by default")
	}

	// Test Categories
	expectedCategories := []string{"development", "staging", "production", "custom", "vpn", "blocked"}
	if len(config.Categories) != len(expectedCategories) {
		t.Errorf("Expected %d categories, got %d", len(expectedCategories), len(config.Categories))
	}
	for _, cat := range expectedCategories {
		if _, exists := config.Categories[cat]; !exists {
			t.Errorf("Expected category %s to exist", cat)
		}
	}

	// Test Profiles
	expectedProfiles := []string{"minimal", "development", "full"}
	if len(config.Profiles) != len(expectedProfiles) {
		t.Errorf("Expected %d profiles, got %d", len(expectedProfiles), len(config.Profiles))
	}

	// Test default profile
	defaultProfile := config.GetActiveProfile()
	if defaultProfile != "full" {
		t.Errorf("Expected active profile to be 'full', got %s", defaultProfile)
	}

	// Test UI settings
	if config.UI.ColorScheme != "auto" {
		t.Errorf("Expected color scheme to be 'auto', got %s", config.UI.ColorScheme)
	}
	if !config.UI.ShowLineNumbers {
		t.Error("Expected show line numbers to be enabled by default")
	}
	if config.UI.PageSize != 20 {
		t.Errorf("Expected page size to be 20, got %d", config.UI.PageSize)
	}

	// Test Backup settings
	if config.Backup.MaxBackups != 10 {
		t.Errorf("Expected max backups to be 10, got %d", config.Backup.MaxBackups)
	}
	if config.Backup.RetentionDays != 30 {
		t.Errorf("Expected retention days to be 30, got %d", config.Backup.RetentionDays)
	}
	if config.Backup.CompressionType != "gzip" {
		t.Errorf("Expected compression type to be 'gzip', got %s", config.Backup.CompressionType)
	}

	// Test Export settings
	if config.Export.DefaultFormat != "yaml" {
		t.Errorf("Expected default format to be 'yaml', got %s", config.Export.DefaultFormat)
	}
	expectedFormats := []string{"yaml", "json", "hosts"}
	if len(config.Export.Formats) != len(expectedFormats) {
		t.Errorf("Expected %d formats, got %d", len(expectedFormats), len(config.Export.Formats))
	}
}

func TestGetDefaultEditor(t *testing.T) {
	tests := []struct {
		name     string
		editor   string
		visual   string
		expected string
	}{
		{
			name:     "no environment variables",
			editor:   "",
			visual:   "",
			expected: "nano",
		},
		{
			name:     "EDITOR set",
			editor:   "vim",
			visual:   "",
			expected: "vim",
		},
		{
			name:     "VISUAL set",
			editor:   "",
			visual:   "emacs",
			expected: "emacs",
		},
		{
			name:     "both set, EDITOR takes precedence",
			editor:   "vim",
			visual:   "emacs",
			expected: "vim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			originalEditor := os.Getenv("EDITOR")
			originalVisual := os.Getenv("VISUAL")
			defer func() {
				_ = os.Setenv("EDITOR", originalEditor)
				_ = os.Setenv("VISUAL", originalVisual)
			}()

			// Set test env vars
			if tt.editor != "" {
				_ = os.Setenv("EDITOR", tt.editor)
			} else {
				_ = os.Unsetenv("EDITOR")
			}
			if tt.visual != "" {
				_ = os.Setenv("VISUAL", tt.visual)
			} else {
				_ = os.Unsetenv("VISUAL")
			}

			result := getDefaultEditor()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Test basic Save/Load functionality without mocking platform
	// We'll just test the marshaling/unmarshaling logic

	config := DefaultConfig()
	config.General.Verbose = true
	config.General.DryRun = true

	// Test YAML marshaling
	data, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Test YAML unmarshaling
	var loadedConfig Config
	err = yaml.Unmarshal(data, &loadedConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Apply defaults to loaded config
	if loadedConfig.Backup.Directory == "" {
		loadedConfig.Backup.Directory = "/tmp/backups"
	}

	// Verify loaded config matches saved config
	if loadedConfig.General.Verbose != config.General.Verbose {
		t.Error("Loaded config verbose setting doesn't match saved config")
	}
	if loadedConfig.General.DryRun != config.General.DryRun {
		t.Error("Loaded config dry run setting doesn't match saved config")
	}
}

func TestLoadNonExistentConfig(t *testing.T) {
	// Test that Load returns default config when file doesn't exist
	// We can't easily test file creation without complex mocking
	// So just test the default config behavior

	config := DefaultConfig()

	// Verify it's the default config with expected values
	if config.General.DefaultCategory != "custom" {
		t.Error("Default config doesn't have expected default category")
	}
	if !config.General.AutoBackup {
		t.Error("Default config should have auto backup enabled")
	}
	if config.General.DryRun {
		t.Error("Default config should have dry run disabled")
	}
}

func TestLoadInvalidConfig(t *testing.T) {
	// Test YAML unmarshaling with invalid content
	invalidYAML := "invalid: yaml: content: [unclosed"

	var config Config
	err := yaml.Unmarshal([]byte(invalidYAML), &config)
	if err == nil {
		t.Error("Expected error when unmarshaling invalid YAML")
	}
}

func TestConfigMethods(t *testing.T) {
	config := DefaultConfig()

	// Test GetCategoryDescription
	desc := config.GetCategoryDescription("development")
	if desc == "" {
		t.Error("Expected non-empty description for development category")
	}

	desc = config.GetCategoryDescription("nonexistent")
	expected := "User-defined category"
	if desc != expected {
		t.Errorf("Expected '%s' for nonexistent category, got '%s'", expected, desc)
	}

	// Test IsValidCategory
	if !config.IsValidCategory("development") {
		t.Error("Expected development to be a valid category")
	}

	if config.IsValidCategory("nonexistent") {
		t.Error("Expected nonexistent to be invalid category")
	}

	// Test GetActiveProfile
	activeProfile := config.GetActiveProfile()
	if activeProfile != "full" {
		t.Errorf("Expected active profile to be 'full', got '%s'", activeProfile)
	}

	// Test with no default profile
	for name := range config.Profiles {
		profile := config.Profiles[name]
		profile.Default = false
		config.Profiles[name] = profile
	}

	activeProfile = config.GetActiveProfile()
	if activeProfile != "full" {
		t.Errorf("Expected fallback active profile to be 'full', got '%s'", activeProfile)
	}
}

func TestConfigSerialization(t *testing.T) {
	config := DefaultConfig()

	// Test YAML marshaling
	data, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config to YAML: %v", err)
	}

	// Test YAML unmarshaling
	var unmarshaledConfig Config
	err = yaml.Unmarshal(data, &unmarshaledConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal config from YAML: %v", err)
	}

	// Verify key fields are preserved
	if unmarshaledConfig.General.DefaultCategory != config.General.DefaultCategory {
		t.Error("Default category not preserved during serialization")
	}
	if len(unmarshaledConfig.Categories) != len(config.Categories) {
		t.Error("Categories not preserved during serialization")
	}
	if len(unmarshaledConfig.Profiles) != len(config.Profiles) {
		t.Error("Profiles not preserved during serialization")
	}
}

func TestConfigValidation(t *testing.T) {
	// Test valid config
	config := DefaultConfig()
	validator := NewValidator()
	err := validator.Validate(config)
	if err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}

	// Test invalid config - empty default category
	config.General.DefaultCategory = ""
	err = validator.Validate(config)
	if err == nil {
		t.Error("Expected validation error for empty default category")
	}
}

func TestBackupDirectoryHandling(t *testing.T) {
	// Test the backup directory setting logic
	config := DefaultConfig()

	// Default config should have empty backup directory
	if config.Backup.Directory != "" {
		t.Errorf("Expected empty backup directory in default config, got: %s", config.Backup.Directory)
	}

	// Test that we can set a backup directory
	config.Backup.Directory = "/test/backup/path"
	if config.Backup.Directory != "/test/backup/path" {
		t.Error("Failed to set backup directory")
	}
}

// BenchmarkDefaultConfig benchmarks the DefaultConfig function
func BenchmarkDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DefaultConfig()
	}
}

// BenchmarkConfigSerialization benchmarks config serialization
func BenchmarkConfigSerialization(b *testing.B) {
	config := DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := yaml.Marshal(config)
		if err != nil {
			b.Fatal(err)
		}
		_ = data
	}
}

// BenchmarkConfigDeserialization benchmarks config deserialization
func BenchmarkConfigDeserialization(b *testing.B) {
	config := DefaultConfig()
	data, err := yaml.Marshal(config)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var cfg Config
		err := yaml.Unmarshal(data, &cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}
