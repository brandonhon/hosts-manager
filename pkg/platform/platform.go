package platform

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

type Platform struct {
	OS       string
	HostsDir string
}

func New() *Platform {
	return &Platform{
		OS:       runtime.GOOS,
		HostsDir: getHostsPath(),
	}
}

func getHostsPath() string {
	switch runtime.GOOS {
	case "windows":
		return `C:\Windows\System32\drivers\etc\hosts`
	case "darwin", "linux":
		return "/etc/hosts"
	default:
		return "/etc/hosts"
	}
}

func (p *Platform) GetHostsFilePath() string {
	return p.HostsDir
}

func (p *Platform) HasWritePermission() bool {
	file, err := os.OpenFile(p.HostsDir, os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

// ElevateIfNeeded requires that the caller is actually running elevated before
// a command modifies the hosts file.
//
// It used to accept mere write permission on the file, so on a system where
// /etc/hosts had been made group-writable the tool would edit it without sudo.
// Requiring elevation makes the privilege boundary the same everywhere rather
// than a property of one file's mode, and it is what the tool is documented to
// need. A dry run bypasses this entirely — see requireHostsWriteAccess.
func (p *Platform) ElevateIfNeeded() error {
	if !p.IsElevated() {
		return p.elevationRequiredError()
	}

	// Elevated but still unable to write: a different problem, and worth
	// saying so rather than repeating "run with sudo" at someone who did.
	if !p.HasWritePermission() {
		return fmt.Errorf("running elevated but still cannot write to hosts file at %s - check file permissions, immutable flags, or disk space", p.HostsDir)
	}

	return nil
}

func (p *Platform) elevationRequiredError() error {
	switch runtime.GOOS {
	case "windows":
		return fmt.Errorf("administrator privileges required to modify hosts file. Please run this command in an elevated Command Prompt or PowerShell")
	case "darwin", "linux":
		return fmt.Errorf("root privileges required to modify hosts file. Please run: sudo %s", strings.Join(os.Args, " "))
	default:
		return fmt.Errorf("elevated privileges required to modify hosts file at %s", p.HostsDir)
	}
}

// invokingUser returns the home directory of the person who ran the command,
// and whether they reached it through sudo.
//
// $HOME is not that person under sudo: most distributions reset it to root's,
// so config, backups and audit logs written while elevated landed in /root and
// were invisible to the user who owned them — and every command needs sudo, so
// that was most of them. SUDO_USER names the account that invoked sudo.
func invokingUser() (home string, viaSudo bool) {
	if name := os.Getenv("SUDO_USER"); name != "" {
		if u, err := user.Lookup(name); err == nil && u.HomeDir != "" {
			return u.HomeDir, true
		}
	}
	return os.Getenv("HOME"), false
}

// xdgDir returns the named XDG directory, or "" when it should not be trusted.
//
// Run normally it is the caller's and used as-is. Under sudo it may be unset,
// may still hold root's value, or -- with sudo -E -- may be the invoking
// user's. Accepting it only when it resolves inside that user's home tells the
// three apart without having to guess.
func xdgDir(name, home string, viaSudo bool) string {
	value := os.Getenv(name)
	if value == "" {
		return ""
	}
	if !viaSudo {
		return value
	}
	if home == "" {
		return ""
	}
	if rel, err := filepath.Rel(home, value); err == nil && !strings.HasPrefix(rel, "..") {
		return value
	}
	return ""
}

func (p *Platform) IsElevated() bool {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("net", "session")
		return cmd.Run() == nil
	case "darwin", "linux":
		return os.Geteuid() == 0
	default:
		return false
	}
}

// GetConfigDir returns the configuration directory for the invoking user,
// which under sudo is the account that ran sudo rather than root.
//
// XDG_CONFIG_HOME is honored on macOS as well as Linux. macOS previously used
// the XDG *default* path, ~/.config, while ignoring the variable that is
// supposed to override it — the worst of both conventions. Someone who sets
// the variable has stated an intent; the platform default applies when they
// have not.
func (p *Platform) GetConfigDir() string {
	home, viaSudo := invokingUser()

	switch runtime.GOOS {
	case "windows":
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return appdata + `\hosts-manager`
		}
		return `C:\ProgramData\hosts-manager`
	case "darwin", "linux":
		if xdg := xdgDir("XDG_CONFIG_HOME", home, viaSudo); xdg != "" {
			return filepath.Join(xdg, "hosts-manager")
		}
		if home != "" {
			return filepath.Join(home, ".config", "hosts-manager")
		}
		return "/etc/hosts-manager"
	default:
		return "/etc/hosts-manager"
	}
}

// GetDataDir returns the data directory — backups and audit logs — for the
// invoking user, which under sudo is the account that ran sudo rather than
// root.
//
// XDG_DATA_HOME is honored on macOS too. Unlike the config directory the
// defaults still differ by platform: ~/Library/Application Support is the macOS
// convention and stays the default there, and ~/.local/share on Linux. These belong with the person using the tool: it is a single-user
// utility, and every writing command needs sudo, so anchoring them to root
// would have put nearly all of them out of reach.
func (p *Platform) GetDataDir() string {
	home, viaSudo := invokingUser()

	switch runtime.GOOS {
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return localAppData + `\hosts-manager`
		}
		return p.GetConfigDir()
	case "darwin":
		if xdg := xdgDir("XDG_DATA_HOME", home, viaSudo); xdg != "" {
			return filepath.Join(xdg, "hosts-manager")
		}
		if home != "" {
			return filepath.Join(home, "Library", "Application Support", "hosts-manager")
		}
		return p.GetConfigDir()
	case "linux":
		if xdg := xdgDir("XDG_DATA_HOME", home, viaSudo); xdg != "" {
			return filepath.Join(xdg, "hosts-manager")
		}
		if home != "" {
			return filepath.Join(home, ".local", "share", "hosts-manager")
		}
		return p.GetConfigDir()
	default:
		return p.GetConfigDir()
	}
}
