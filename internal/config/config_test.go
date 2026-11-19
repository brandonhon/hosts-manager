package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brandonhon/hosts-manager/internal/platform"
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

	// Test Profiles
	expectedProfiles := []string{"minimal", "development", "full"}
	if len(config.Profiles) != len(expectedProfiles) {
		t.Errorf("Expected %d profiles, got %d", len(expectedProfiles), len(config.Profiles))
	}

	// The "full" profile is the one marked default.
	if !config.Profiles["full"].Default {
		t.Error("Expected the \"full\" profile to be the default")
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
	if len(unmarshaledConfig.Profiles) != len(config.Profiles) {
		t.Error("Profiles not preserved during serialization")
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

// isolateConfigDirs points every platform's config and data directory lookup
// at temporary directories so Load and Save do not touch the real user config.
func isolateConfigDirs(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", // unix
		"APPDATA", "LOCALAPPDATA", // windows
	} {
		t.Setenv(key, t.TempDir())
	}
}

// TestLoadIgnoresRemovedKeys guards upgrades: config files written by older
// versions carry ui, categories, editor and export.formats sections that no
// longer exist, and must still load.
func TestLoadIgnoresRemovedKeys(t *testing.T) {
	isolateConfigDirs(t)
	dir := platform.New().GetConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	old := `general:
  default_category: custom
  auto_backup: true
  editor: nvim
categories:
  custom: User-defined entries
profiles:
  full: {description: all, categories: [custom], default: true}
ui:
  color_scheme: auto
  page_size: 20
  key_bindings: {quit: q}
backup: {max_backups: 10, retention_days: 30, compression_type: gzip}
export:
  default_format: yaml
  formats:
    yaml: {extension: .yaml, template: "{{.}}"}
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(old), 0600); err != nil {
		t.Fatal(err)
	}

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() of an older config file failed: %v", err)
	}
	if config.General.DefaultCategory != "custom" || !config.Profiles["full"].Default {
		t.Errorf("kept settings not loaded: %+v", config)
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

// TestLoadFirstRunResolvesBackupDirectory guards the first-run gap: the backup
// directory default was only applied on the read path, so the very first Load
// returned an empty one and every command with AutoBackup enabled — the
// default — failed at os.MkdirAll("").
func TestLoadFirstRunResolvesBackupDirectory(t *testing.T) {
	isolateConfigDirs(t)

	config, err := Load()
	if err != nil {
		t.Fatalf("first Load(): %v", err)
	}
	if config.Backup.Directory == "" {
		t.Fatal("first run returned an empty backup directory")
	}
	if !strings.HasSuffix(filepath.Clean(config.Backup.Directory), filepath.Join("hosts-manager", "backups")) {
		t.Errorf("backup directory = %q, want it under the data dir", config.Backup.Directory)
	}

	// The directory the first run reports must actually be usable, which is
	// what the original bug broke.
	if err := os.MkdirAll(config.Backup.Directory, 0700); err != nil {
		t.Errorf("backup directory is not creatable: %v", err)
	}

	// The second run must agree with the first.
	second, err := Load()
	if err != nil {
		t.Fatalf("second Load(): %v", err)
	}
	if second.Backup.Directory != config.Backup.Directory {
		t.Errorf("second run resolved %q, first resolved %q", second.Backup.Directory, config.Backup.Directory)
	}
}

// TestLoadFirstRunLeavesBackupDirectoryEmptyOnDisk pins the deliberate choice
// that the written config keeps an empty backup directory: empty means "derive
// it from the data dir at load time", which survives the home directory moving.
func TestLoadFirstRunLeavesBackupDirectoryEmptyOnDisk(t *testing.T) {
	isolateConfigDirs(t)

	if _, err := Load(); err != nil {
		t.Fatalf("Load(): %v", err)
	}

	data, err := os.ReadFile(filepath.Join(platform.New().GetConfigDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("config file was not written: %v", err)
	}
	var onDisk Config
	if err := yaml.Unmarshal(data, &onDisk); err != nil {
		t.Fatal(err)
	}
	if onDisk.Backup.Directory != "" {
		t.Errorf("config on disk pinned an absolute backup directory %q; it should stay empty",
			onDisk.Backup.Directory)
	}
}
