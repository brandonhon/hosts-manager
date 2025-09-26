package platform

import (
	"fmt"
	"os"
	"os/exec"
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

func (p *Platform) NeedsElevation() bool {
	return true
}

func (p *Platform) HasWritePermission() bool {
	file, err := os.OpenFile(p.HostsDir, os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

func (p *Platform) ElevateIfNeeded() error {
	if p.HasWritePermission() {
		return nil
	}

	// Check if already elevated but still no write permission (other issue)
	if p.IsElevated() {
		return fmt.Errorf("elevated privileges detected but still cannot write to hosts file at %s - check file permissions or disk space", p.HostsDir)
	}

	switch runtime.GOOS {
	case "windows":
		return fmt.Errorf("Administrator privileges required to modify hosts file. Please run this command in an elevated Command Prompt or PowerShell")
	case "darwin", "linux":
		return fmt.Errorf("root privileges required to modify hosts file. Please run: sudo %s", strings.Join(os.Args, " "))
	default:
		return fmt.Errorf("insufficient permissions to modify hosts file at %s", p.HostsDir)
	}
}

// ElevateIfNeededStrict performs stricter privilege checking for security-sensitive operations
func (p *Platform) ElevateIfNeededStrict() error {
	// For security-sensitive operations, we should always check for proper elevation
	// even if the file happens to be writable by regular users
	if !p.IsElevated() {
		switch runtime.GOOS {
		case "windows":
			return fmt.Errorf("Administrator privileges required for this security-sensitive operation. Please run this command in an elevated Command Prompt or PowerShell")
		case "darwin", "linux":
			return fmt.Errorf("root privileges required for this security-sensitive operation. Please run: sudo %s", strings.Join(os.Args, " "))
		default:
			return fmt.Errorf("elevated privileges required for this security-sensitive operation")
		}
	}

	if !p.HasWritePermission() {
		return fmt.Errorf("cannot write to hosts file at %s - check file permissions or disk space", p.HostsDir)
	}

	return nil
}

func (p *Platform) CreateBackupPath(timestamp string) string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf(`C:\Windows\System32\drivers\etc\hosts.backup.%s`, timestamp)
	default:
		return fmt.Sprintf("/etc/hosts.backup.%s", timestamp)
	}
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

func (p *Platform) GetConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return appdata + `\hosts-manager`
		}
		return `C:\ProgramData\hosts-manager`
	case "darwin":
		if home := os.Getenv("HOME"); home != "" {
			return home + "/.config/hosts-manager"
		}
		return "/etc/hosts-manager"
	case "linux":
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			return xdgConfig + "/hosts-manager"
		}
		if home := os.Getenv("HOME"); home != "" {
			return home + "/.config/hosts-manager"
		}
		return "/etc/hosts-manager"
	default:
		return "/etc/hosts-manager"
	}
}

func (p *Platform) GetDataDir() string {
	switch runtime.GOOS {
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return localAppData + `\hosts-manager`
		}
		return p.GetConfigDir()
	case "darwin":
		if home := os.Getenv("HOME"); home != "" {
			return home + "/Library/Application Support/hosts-manager"
		}
		return p.GetConfigDir()
	case "linux":
		if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
			return xdgData + "/hosts-manager"
		}
		if home := os.Getenv("HOME"); home != "" {
			return home + "/.local/share/hosts-manager"
		}
		return p.GetConfigDir()
	default:
		return p.GetConfigDir()
	}
}

func (p *Platform) SanitizePath(path string) string {
	switch runtime.GOOS {
	case "windows":
		return strings.ReplaceAll(path, "/", "\\")
	default:
		return strings.ReplaceAll(path, "\\", "/")
	}
}