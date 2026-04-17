package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidateFilePath validates a file path for security concerns
func ValidateFilePath(path string) error {
	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if path is within allowed directories
	homeDir, _ := os.UserHomeDir()
	tmpDir := os.TempDir()

	allowedPrefixes := []string{
		homeDir,
		tmpDir,
		"/tmp",
		"/var/tmp",
	}

	// On Windows, also allow common temp directories
	if os.PathSeparator == '\\' {
		allowedPrefixes = append(allowedPrefixes,
			os.Getenv("TEMP"),
			os.Getenv("TMP"),
		)
	}

	allowed := false
	for _, prefix := range allowedPrefixes {
		if prefix != "" && strings.HasPrefix(absPath, prefix) {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("path not in allowed directories: %s", absPath)
	}

	return nil
}

// IsValidEditor checks if an editor command is in the allowed list
func IsValidEditor(editor string) bool {
	allowedEditors := map[string]bool{
		"vim":     true,
		"vi":      true,
		"nano":    true,
		"emacs":   true,
		"code":    true,
		"subl":    true,
		"notepad": true,
		"gedit":   true,
	}

	// Extract just the command name (without path or args)
	editorCmd := filepath.Base(editor)
	editorCmd = strings.TrimSuffix(editorCmd, ".exe")
	editorCmd = strings.Split(editorCmd, " ")[0]

	return allowedEditors[editorCmd]
}

// RunCommand executes a command safely
func RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ConfirmDestructive prompts the user for confirmation before destructive operations
func ConfirmDestructive(message string) (bool, error) {
	fmt.Printf("%s [y/N]: ", message)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// ConfirmWithDefault prompts the user with a default option
func ConfirmWithDefault(message string, defaultYes bool) (bool, error) {
	prompt := "[y/N]"
	if defaultYes {
		prompt = "[Y/n]"
	}

	fmt.Printf("%s %s: ", message, prompt)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" {
		return defaultYes, nil
	}

	return response == "y" || response == "yes", nil
}

// ValidateCategoryName validates a category name
func ValidateCategoryName(name string) error {
	if name == "" {
		return fmt.Errorf("category name cannot be empty")
	}

	if len(name) > 50 {
		return fmt.Errorf("category name too long (max 50 characters)")
	}

	// Category names should be alphanumeric with hyphens and underscores
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("category name must contain only alphanumeric characters, hyphens, and underscores")
	}

	// Reserved category names
	reserved := map[string]bool{
		"uncategorized": true,
		"default":       true,
		"system":        true,
	}

	if reserved[strings.ToLower(name)] {
		return fmt.Errorf("category name '%s' is reserved", name)
	}

	return nil
}

// FormatEntryLine formats an entry line for display
func FormatEntryLine(ip string, hostnames []string, comment string, disabled bool) string {
	prefix := ""
	if disabled {
		prefix = "# "
	}

	line := fmt.Sprintf("%s%-15s %s", prefix, ip, strings.Join(hostnames, " "))

	if comment != "" {
		line += fmt.Sprintf("  # %s", comment)
	}

	return line
}

// ParseHostnameList parses a comma or space-separated list of hostnames
func ParseHostnameList(input string) []string {
	// Replace commas with spaces
	input = strings.ReplaceAll(input, ",", " ")

	// Split on whitespace
	parts := strings.Fields(input)

	// Remove duplicates and empty strings
	seen := make(map[string]bool)
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && !seen[part] {
			seen[part] = true
			result = append(result, part)
		}
	}

	return result
}

// TruncateString truncates a string to maxLen with ellipsis
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return s[:maxLen]
	}

	return s[:maxLen-3] + "..."
}

// PadRight pads a string with spaces to the right
func PadRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

// PadLeft pads a string with spaces to the left
func PadLeft(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return strings.Repeat(" ", length-len(s)) + s
}

// FindEditor finds the best available editor
func FindEditor() string {
	// Check EDITOR environment variable
	if editor := os.Getenv("EDITOR"); editor != "" && IsValidEditor(editor) {
		return editor
	}

	// Check VISUAL environment variable
	if visual := os.Getenv("VISUAL"); visual != "" && IsValidEditor(visual) {
		return visual
	}

	// Try common editors in order of preference
	editors := []string{"vim", "vi", "nano", "emacs", "code", "notepad"}

	for _, editor := range editors {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}

	// Fallback
	if os.PathSeparator == '\\' {
		return "notepad"
	}
	return "vi"
}

// IsInteractive checks if the program is running in an interactive terminal
func IsInteractive() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ErrorWithSuggestion returns an error with a helpful suggestion
func ErrorWithSuggestion(err error, suggestion string) error {
	return fmt.Errorf("%w\n\nSuggestion: %s", err, suggestion)
}
