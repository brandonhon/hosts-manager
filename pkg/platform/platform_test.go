package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	p := New()
	if p == nil {
		t.Fatal("New() returned nil")
	}
	if p.OS != runtime.GOOS {
		t.Errorf("New().OS = %v, want %v", p.OS, runtime.GOOS)
	}
}

func TestGetHostsFilePath(t *testing.T) {
	tests := []struct {
		name         string
		expectedPath string
	}{
		{
			name: "default hosts file path",
			expectedPath: func() string {
				if runtime.GOOS == "windows" {
					return `C:\Windows\System32\drivers\etc\hosts`
				}
				return "/etc/hosts"
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			path := p.GetHostsFilePath()

			if path != tt.expectedPath {
				t.Errorf("GetHostsFilePath() = %v, want %v", path, tt.expectedPath)
			}
		})
	}
}

func TestGetConfigDir(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv func()
		cleanup  func()
		validate func(string) bool
	}{
		{
			name: "default config directory",
			setupEnv: func() {
				_ = os.Unsetenv("XDG_CONFIG_HOME")
				_ = os.Unsetenv("APPDATA")
			},
			cleanup: func() {},
			validate: func(path string) bool {
				return strings.Contains(path, "hosts-manager") &&
					(strings.Contains(path, ".config") ||
						strings.Contains(path, "AppData") ||
						strings.Contains(path, "ProgramData") ||
						strings.Contains(path, "/etc/"))
			},
		},
		{
			name: "custom XDG_CONFIG_HOME",
			setupEnv: func() {
				if runtime.GOOS != "windows" {
					_ = os.Setenv("XDG_CONFIG_HOME", "/tmp/custom-config")
				}
			},
			cleanup: func() {
				_ = os.Unsetenv("XDG_CONFIG_HOME")
			},
			validate: func(path string) bool {
				if runtime.GOOS == "windows" {
					return true // Skip this test on Windows
				}
				// On Darwin, XDG_CONFIG_HOME is not used by default - it uses ~/.config
				if runtime.GOOS == "darwin" {
					return strings.Contains(path, ".config/hosts-manager")
				}
				return strings.HasPrefix(path, "/tmp/custom-config")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanup()

			p := New()
			configDir := p.GetConfigDir()

			if configDir == "" {
				t.Error("GetConfigDir() returned empty string")
				return
			}

			if !tt.validate(configDir) {
				t.Errorf("GetConfigDir() = %v, failed validation", configDir)
			}
		})
	}
}

func TestGetDataDir(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv func()
		cleanup  func()
		validate func(string) bool
	}{
		{
			name: "default data directory",
			setupEnv: func() {
				_ = os.Unsetenv("XDG_DATA_HOME")
				_ = os.Unsetenv("LOCALAPPDATA")
			},
			cleanup: func() {},
			validate: func(path string) bool {
				switch runtime.GOOS {
				case "windows":
					return strings.Contains(path, "hosts-manager")
				case "darwin":
					return strings.Contains(path, "Library/Application Support") && strings.Contains(path, "hosts-manager")
				default: // linux
					return strings.Contains(path, ".local/share") && strings.Contains(path, "hosts-manager")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanup()

			p := New()
			dataDir := p.GetDataDir()

			if dataDir == "" {
				t.Error("GetDataDir() returned empty string")
				return
			}

			if !tt.validate(dataDir) {
				t.Errorf("GetDataDir() = %v, failed validation", dataDir)
			}
		})
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(string) bool
	}{
		{
			name:  "clean absolute path",
			input: "/etc/hosts",
			validate: func(result string) bool {
				return strings.Contains(result, "hosts")
			},
		},
		{
			name:  "relative path gets converted to absolute",
			input: "hosts",
			validate: func(result string) bool {
				return filepath.IsAbs(result) && strings.Contains(result, "hosts")
			},
		},
		{
			name:  "current directory",
			input: ".",
			validate: func(result string) bool {
				return filepath.IsAbs(result)
			},
		},
		{
			name:  "path with unresolved traversal should trigger fallback",
			input: "..",
			validate: func(result string) bool {
				return strings.Contains(result, "safe_fallback")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result := p.SanitizePath(tt.input)

			if !tt.validate(result) {
				t.Errorf("SanitizePath(%q) = %q, failed validation", tt.input, result)
			}
		})
	}
}

func TestSanitizePathWithInfo(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectViolation bool
		validatePath    func(string) bool
	}{
		{
			name:            "clean path",
			input:           "/etc/hosts",
			expectViolation: false,
			validatePath: func(path string) bool {
				return strings.Contains(path, "hosts")
			},
		},
		{
			name:            "path traversal attack with unresolved dots",
			input:           "../../../sensitive",
			expectViolation: true,
			validatePath: func(path string) bool {
				return strings.Contains(path, "safe_fallback")
			},
		},
		{
			name:            "null byte attack",
			input:           "/etc/hosts\x00malicious",
			expectViolation: true,
			validatePath: func(path string) bool {
				return strings.Contains(path, "safe_fallback")
			},
		},
		{
			name:            "unicode control characters",
			input:           "/etc/hosts\u0000\u0001",
			expectViolation: true,
			validatePath: func(path string) bool {
				return strings.Contains(path, "safe_fallback")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result, info := p.SanitizePathWithInfo(tt.input)

			if !tt.validatePath(result) {
				t.Errorf("SanitizePathWithInfo(%q) path = %q, failed validation", tt.input, result)
			}

			if tt.expectViolation && info == nil {
				t.Errorf("SanitizePathWithInfo(%q) expected security violation info, got nil", tt.input)
			} else if !tt.expectViolation && info != nil {
				t.Errorf("SanitizePathWithInfo(%q) unexpected security violation info: %+v", tt.input, info)
			}

			if info != nil {
				if info.OriginalPath != tt.input {
					t.Errorf("SecurityInfo.OriginalPath = %q, want %q", info.OriginalPath, tt.input)
				}
				if info.Violation == "" {
					t.Error("SecurityInfo.Violation is empty")
				}
			}
		})
	}
}

func TestNeedsElevation(t *testing.T) {
	p := New()
	needsElevation := p.NeedsElevation()

	// Should always return true for this implementation
	if !needsElevation {
		t.Error("NeedsElevation() should return true")
	}
}

func TestHasWritePermission(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "check hosts file write permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			canWrite := p.HasWritePermission()

			// We can't assert a specific value since it depends on current permissions
			// Just ensure the method doesn't panic and returns a boolean
			t.Logf("HasWritePermission() = %v", canWrite)
		})
	}
}

func TestElevateIfNeeded(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping elevation test in short mode")
	}

	tests := []struct {
		name        string
		skipOnOS    string
		expectError bool
	}{
		{
			name:        "elevation check",
			expectError: false, // In test environment, we expect this to work or gracefully handle
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnOS != "" && runtime.GOOS == tt.skipOnOS {
				t.Skipf("Skipping test on %s", tt.skipOnOS)
			}

			p := New()
			err := p.ElevateIfNeeded()

			if tt.expectError && err == nil {
				t.Error("ElevateIfNeeded() expected error, got nil")
			} else if !tt.expectError && err != nil {
				// In test environment, elevation might fail - this is acceptable
				t.Logf("ElevateIfNeeded() returned error (acceptable in test): %v", err)
			}
		})
	}
}

func TestElevateIfNeededStrict(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping strict elevation test in short mode")
	}

	p := New()
	err := p.ElevateIfNeededStrict()

	// In test environment, this might fail - that's acceptable
	t.Logf("ElevateIfNeededStrict() error: %v", err)
}

func TestIsElevated(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "check elevation status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			elevated := p.IsElevated()

			// We can't assert a specific value since it depends on test execution context
			// Just ensure the method doesn't panic and returns a boolean
			t.Logf("IsElevated() = %v", elevated)
		})
	}
}

func TestCreateBackupPath(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
		validate  func(string) bool
	}{
		{
			name:      "valid timestamp",
			timestamp: "20240101-120000",
			validate: func(path string) bool {
				return strings.Contains(path, "20240101-120000") && strings.Contains(path, "backup")
			},
		},
		{
			name:      "empty timestamp",
			timestamp: "",
			validate: func(path string) bool {
				return strings.Contains(path, "backup")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			backupPath := p.CreateBackupPath(tt.timestamp)

			if backupPath == "" {
				t.Error("CreateBackupPath() returned empty string")
				return
			}

			if !tt.validate(backupPath) {
				t.Errorf("CreateBackupPath(%q) = %q, failed validation", tt.timestamp, backupPath)
			}
		})
	}
}

// Benchmark tests
func BenchmarkSanitizePath(b *testing.B) {
	p := New()
	testPath := "/etc/../tmp/../../etc/hosts"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.SanitizePath(testPath)
	}
}

func BenchmarkSanitizePathWithInfo(b *testing.B) {
	p := New()
	testPath := "/etc/../tmp/../../etc/hosts\x00malicious"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.SanitizePathWithInfo(testPath)
	}
}

// Edge case tests
func TestSanitizePathEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(string, *PathSecurityInfo) bool
	}{
		{
			name:  "empty string",
			input: "",
			validate: func(path string, info *PathSecurityInfo) bool {
				return filepath.IsAbs(path) && info == nil
			},
		},
		{
			name:  "single dot",
			input: ".",
			validate: func(path string, info *PathSecurityInfo) bool {
				return filepath.IsAbs(path) && info == nil
			},
		},
		{
			name:  "double dot",
			input: "..",
			validate: func(path string, info *PathSecurityInfo) bool {
				return strings.Contains(path, "safe_fallback") && info != nil && info.Violation == "path_traversal"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result, info := p.SanitizePathWithInfo(tt.input)

			if !tt.validate(result, info) {
				t.Errorf("SanitizePathWithInfo(%q) = (%q, %+v), failed validation", tt.input, result, info)
			}
		})
	}
}
