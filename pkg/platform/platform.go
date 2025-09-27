package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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

// PathSecurityInfo contains information about security issues found during path sanitization
type PathSecurityInfo struct {
	Violation     string                 // Type of violation found
	OriginalPath  string                 // Original unsanitized path  
	Details       map[string]interface{} // Additional violation details
	SanitizedPath string                 // The safe fallback path
}

func (p *Platform) SanitizePath(path string) string {
	result, _ := p.SanitizePathWithInfo(path)
	return result
}

func (p *Platform) SanitizePathWithInfo(path string) (string, *PathSecurityInfo) {
	// Clean the path to resolve any relative components and remove redundant separators
	cleanPath := filepath.Clean(path)
	
	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		// Return a safe default path with process ID and timestamp to prevent races
		timestamp := time.Now().Format("20060102-150405")
		fallbackName := fmt.Sprintf("safe_fallback_%d_%s", os.Getpid(), timestamp)
		safePath := filepath.Join(filepath.Dir(p.GetHostsFilePath()), fallbackName)
		
		return safePath, &PathSecurityInfo{
			Violation:     "path_traversal",
			OriginalPath:  path,
			Details:       map[string]interface{}{"original_path": path, "cleaned_path": cleanPath},
			SanitizedPath: safePath,
		}
	}
	
	// Additional validation for absolute paths to prevent access outside expected directories
	abs, err := filepath.Abs(cleanPath)
	if err != nil {
		// If we can't get absolute path, return safe default with unique identifier
		timestamp := time.Now().Format("20060102-150405")
		fallbackName := fmt.Sprintf("safe_fallback_%d_%s", os.Getpid(), timestamp)
		safePath := filepath.Join(filepath.Dir(p.GetHostsFilePath()), fallbackName)
		
		return safePath, &PathSecurityInfo{
			Violation:     "path_resolution_failed",
			OriginalPath:  path,
			Details:       map[string]interface{}{"error": err.Error()},
			SanitizedPath: safePath,
		}
	}
	
	// Ensure the path doesn't contain null bytes or other dangerous characters
	if strings.ContainsRune(abs, 0) {
		// Return safe fallback
		timestamp := time.Now().Format("20060102-150405")
		fallbackName := fmt.Sprintf("safe_fallback_%d_%s", os.Getpid(), timestamp)
		safePath := filepath.Join(filepath.Dir(p.GetHostsFilePath()), fallbackName)
		
		return safePath, &PathSecurityInfo{
			Violation:     "null_byte_injection",
			OriginalPath:  path,
			Details:       nil,
			SanitizedPath: safePath,
		}
	}
	
	// No security issues found, return sanitized path
	switch runtime.GOOS {
	case "windows":
		return strings.ReplaceAll(abs, "/", "\\"), nil
	default:
		return strings.ReplaceAll(abs, "\\", "/"), nil
	}
}