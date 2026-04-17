package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brandonhon/hosts-manager/internal/config"
	"github.com/brandonhon/hosts-manager/internal/hosts"
	"gopkg.in/yaml.v3"
)

// TestContext provides isolated testing environment for CLI commands
type TestContext struct {
	// TempDir is the root temporary directory for this test
	TempDir string

	// HostsFile is the path to the test hosts file
	HostsFile string

	// ConfigDir is the path to the test config directory
	ConfigDir string

	// ConfigFile is the path to the test config file
	ConfigFile string

	// BackupDir is the path to the test backup directory
	BackupDir string

	// AuditLogFile is the path to the test audit log
	AuditLogFile string

	// T is the testing context
	T *testing.T
}

// SetupTestEnvironment creates an isolated test environment with temporary directories
func SetupTestEnvironment(t *testing.T) *TestContext {
	t.Helper()

	// Create temporary directory
	tempDir := t.TempDir()

	ctx := &TestContext{
		TempDir:      tempDir,
		HostsFile:    filepath.Join(tempDir, "hosts"),
		ConfigDir:    filepath.Join(tempDir, "config"),
		ConfigFile:   filepath.Join(tempDir, "config", "config.yaml"),
		BackupDir:    filepath.Join(tempDir, "backups"),
		AuditLogFile: filepath.Join(tempDir, "audit.log"),
		T:            t,
	}

	// Create necessary directories
	if err := os.MkdirAll(ctx.ConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	if err := os.MkdirAll(ctx.BackupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create default hosts file
	ctx.WriteHostsFile(ctx.DefaultHostsContent())

	// Create default config
	ctx.WriteConfig(config.DefaultConfig())

	return ctx
}

// DefaultHostsContent returns a default hosts file content for testing
func (ctx *TestContext) DefaultHostsContent() string {
	return `# Default hosts file for testing
127.0.0.1       localhost
::1             localhost

# @category development
127.0.0.1       dev.local
192.168.1.100   api.dev

# @category staging
10.0.1.50       staging.local

# @category custom Custom category
172.16.0.10     custom.local
`
}

// WriteHostsFile writes content to the test hosts file
func (ctx *TestContext) WriteHostsFile(content string) {
	ctx.T.Helper()

	if err := os.WriteFile(ctx.HostsFile, []byte(content), 0644); err != nil {
		ctx.T.Fatalf("Failed to write hosts file: %v", err)
	}
}

// ReadHostsFile reads the test hosts file content
func (ctx *TestContext) ReadHostsFile() string {
	ctx.T.Helper()

	content, err := os.ReadFile(ctx.HostsFile)
	if err != nil {
		ctx.T.Fatalf("Failed to read hosts file: %v", err)
	}

	return string(content)
}

// WriteConfig writes a config to the test config file
func (ctx *TestContext) WriteConfig(cfg *config.Config) {
	ctx.T.Helper()

	// Note: config.Save saves to the platform-specific location
	// For testing, we'll marshal and write directly
	data, err := yaml.Marshal(cfg)
	if err != nil {
		ctx.T.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(ctx.ConfigFile, data, 0644); err != nil {
		ctx.T.Fatalf("Failed to write config: %v", err)
	}
}

// LoadConfig loads the config from the test config file
func (ctx *TestContext) LoadConfig() *config.Config {
	ctx.T.Helper()

	// For testing, use the default config
	// In real usage, config.Load() doesn't take parameters
	cfg := config.DefaultConfig()

	// Try to load from our test config file
	if _, err := os.Stat(ctx.ConfigFile); err == nil {
		data, err := os.ReadFile(ctx.ConfigFile)
		if err != nil {
			ctx.T.Fatalf("Failed to read config: %v", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			ctx.T.Fatalf("Failed to unmarshal config: %v", err)
		}
	}

	return cfg
}

// ParseHostsFile parses the test hosts file
func (ctx *TestContext) ParseHostsFile() *hosts.HostsFile {
	ctx.T.Helper()

	parser := hosts.NewParser(ctx.HostsFile)
	hf, err := parser.Parse()
	if err != nil {
		ctx.T.Fatalf("Failed to parse hosts file: %v", err)
	}

	return hf
}

// WriteHostsFileFromStruct writes a HostsFile struct to the test hosts file
func (ctx *TestContext) WriteHostsFileFromStruct(hf *hosts.HostsFile) {
	ctx.T.Helper()

	if err := hf.Write(ctx.HostsFile); err != nil {
		ctx.T.Fatalf("Failed to write hosts file: %v", err)
	}
}

// AssertHostsFileContains checks if the hosts file contains the given string
func (ctx *TestContext) AssertHostsFileContains(needle string) {
	ctx.T.Helper()

	content := ctx.ReadHostsFile()
	if !contains(content, needle) {
		ctx.T.Errorf("Hosts file does not contain expected string: %q\nContent:\n%s", needle, content)
	}
}

// AssertHostsFileNotContains checks if the hosts file does not contain the given string
func (ctx *TestContext) AssertHostsFileNotContains(needle string) {
	ctx.T.Helper()

	content := ctx.ReadHostsFile()
	if contains(content, needle) {
		ctx.T.Errorf("Hosts file contains unexpected string: %q\nContent:\n%s", needle, content)
	}
}

// AssertEntryExists checks if an entry with the given IP and hostname exists
func (ctx *TestContext) AssertEntryExists(ip, hostname string) {
	ctx.T.Helper()

	hf := ctx.ParseHostsFile()
	found := false

	for _, cat := range hf.Categories {
		for _, entry := range cat.Entries {
			if entry.IP == ip {
				for _, h := range entry.Hostnames {
					if h == hostname {
						found = true
						break
					}
				}
			}
			if found {
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		ctx.T.Errorf("Entry not found: IP=%s, Hostname=%s", ip, hostname)
	}
}

// AssertEntryNotExists checks if an entry with the given IP and hostname does not exist
func (ctx *TestContext) AssertEntryNotExists(ip, hostname string) {
	ctx.T.Helper()

	hf := ctx.ParseHostsFile()
	found := false

	for _, cat := range hf.Categories {
		for _, entry := range cat.Entries {
			if entry.IP == ip {
				for _, h := range entry.Hostnames {
					if h == hostname {
						found = true
						break
					}
				}
			}
			if found {
				break
			}
		}
		if found {
			break
		}
	}

	if found {
		ctx.T.Errorf("Entry should not exist but was found: IP=%s, Hostname=%s", ip, hostname)
	}
}

// AssertEntryEnabled checks if an entry is enabled (not commented)
func (ctx *TestContext) AssertEntryEnabled(ip, hostname string) {
	ctx.T.Helper()

	hf := ctx.ParseHostsFile()
	found := false
	enabled := false

	for _, cat := range hf.Categories {
		for _, entry := range cat.Entries {
			if entry.IP == ip {
				for _, h := range entry.Hostnames {
					if h == hostname {
						found = true
						enabled = entry.Enabled
						break
					}
				}
			}
			if found {
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		ctx.T.Errorf("Entry not found: IP=%s, Hostname=%s", ip, hostname)
		return
	}

	if !enabled {
		ctx.T.Errorf("Entry should be enabled but is disabled: IP=%s, Hostname=%s", ip, hostname)
	}
}

// AssertEntryDisabled checks if an entry is disabled (commented)
func (ctx *TestContext) AssertEntryDisabled(ip, hostname string) {
	ctx.T.Helper()

	hf := ctx.ParseHostsFile()
	found := false
	enabled := true

	for _, cat := range hf.Categories {
		for _, entry := range cat.Entries {
			if entry.IP == ip {
				for _, h := range entry.Hostnames {
					if h == hostname {
						found = true
						enabled = entry.Enabled
						break
					}
				}
			}
			if found {
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		ctx.T.Errorf("Entry not found: IP=%s, Hostname=%s", ip, hostname)
		return
	}

	if enabled {
		ctx.T.Errorf("Entry should be disabled but is enabled: IP=%s, Hostname=%s", ip, hostname)
	}
}

// AssertCategoryExists checks if a category exists
func (ctx *TestContext) AssertCategoryExists(categoryName string) {
	ctx.T.Helper()

	hf := ctx.ParseHostsFile()
	found := false

	for _, cat := range hf.Categories {
		if cat.Name == categoryName {
			found = true
			break
		}
	}

	if !found {
		ctx.T.Errorf("Category not found: %s", categoryName)
	}
}

// AssertBackupExists checks if a backup file exists
func (ctx *TestContext) AssertBackupExists(filename string) {
	ctx.T.Helper()

	backupPath := filepath.Join(ctx.BackupDir, filename)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		ctx.T.Errorf("Backup file does not exist: %s", filename)
	}
}

// CountBackups returns the number of backup files
func (ctx *TestContext) CountBackups() int {
	ctx.T.Helper()

	entries, err := os.ReadDir(ctx.BackupDir)
	if err != nil {
		ctx.T.Fatalf("Failed to read backup directory: %v", err)
	}

	return len(entries)
}

// contains is a helper to check if a string contains a substring
func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || len(needle) == 0 ||
		indexAt(haystack, needle) >= 0)
}

// indexAt finds the index of needle in haystack
func indexAt(haystack, needle string) int {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
