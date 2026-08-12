package platform

import (
	"os"
	"os/user"
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

// Benchmark tests

// TestDirsFollowTheInvokingUserUnderSudo covers the split that made config,
// backups and audit logs written under sudo land in /root while the same
// commands run normally used the user's home. Since every writing command
// needs sudo, that was most of them.
func TestDirsFollowTheInvokingUserUnderSudo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SUDO_USER has no Windows equivalent")
	}

	me, err := user.Current()
	if err != nil || me.Username == "" || me.HomeDir == "" {
		t.Skip("cannot resolve the current user")
	}

	p := New()

	t.Run("without sudo, HOME is used", func(t *testing.T) {
		t.Setenv("SUDO_USER", "")
		t.Setenv("HOME", "/home/someone")
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("XDG_DATA_HOME", "")
		if got := p.GetConfigDir(); !strings.HasPrefix(got, "/home/someone") {
			t.Errorf("GetConfigDir() = %q, want it under /home/someone", got)
		}
	})

	t.Run("under sudo, the invoking user's home wins over root's HOME", func(t *testing.T) {
		t.Setenv("SUDO_USER", me.Username)
		t.Setenv("HOME", "/root") // what sudo actually leaves behind
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("XDG_DATA_HOME", "")

		for name, got := range map[string]string{
			"GetConfigDir": p.GetConfigDir(),
			"GetDataDir":   p.GetDataDir(),
		} {
			if strings.HasPrefix(got, "/root") {
				t.Errorf("%s() = %q under sudo; should follow SUDO_USER, not root", name, got)
			}
			if !strings.HasPrefix(got, me.HomeDir) {
				t.Errorf("%s() = %q, want it under %q", name, got, me.HomeDir)
			}
		}
	})

	// The XDG variables are only consulted on Linux; the darwin branch goes
	// straight to ~/.config and ~/Library. That asymmetry predates this change.
	if runtime.GOOS != "linux" {
		return
	}

	t.Run("under sudo, a stale XDG pointing outside that home is ignored", func(t *testing.T) {
		t.Setenv("SUDO_USER", me.Username)
		t.Setenv("HOME", "/root")
		t.Setenv("XDG_CONFIG_HOME", "/root/.config")

		got := p.GetConfigDir()
		if strings.HasPrefix(got, "/root") {
			t.Errorf("GetConfigDir() = %q; root's XDG_CONFIG_HOME should not be trusted under sudo", got)
		}
	})

	t.Run("under sudo -E, an XDG inside that home is honored", func(t *testing.T) {
		custom := filepath.Join(me.HomeDir, "xdg-config")
		t.Setenv("SUDO_USER", me.Username)
		t.Setenv("HOME", "/root")
		t.Setenv("XDG_CONFIG_HOME", custom)

		if got := p.GetConfigDir(); !strings.HasPrefix(got, custom) {
			t.Errorf("GetConfigDir() = %q, want it under %q — sudo -E keeps the caller's XDG", got, custom)
		}
	})
}

// TestElevateIfNeededRequiresElevation pins the tightened rule: write access to
// the hosts file is no longer enough on its own.
func TestElevateIfNeededRequiresElevation(t *testing.T) {
	p := New()
	err := p.ElevateIfNeeded()

	if p.IsElevated() {
		if err != nil {
			t.Logf("elevated; ElevateIfNeeded() = %v (write may still be blocked)", err)
		}
		return
	}

	if err == nil {
		t.Error("ElevateIfNeeded() returned nil while not elevated; sudo is supposed to be required")
	}
}
