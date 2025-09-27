package config

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}

// ConfigValidator validates configuration values
type ConfigValidator struct {
	errors []ValidationError
}

// NewValidator creates a new configuration validator
func NewValidator() *ConfigValidator {
	return &ConfigValidator{
		errors: make([]ValidationError, 0),
	}
}

// Validate validates the entire configuration
func (v *ConfigValidator) Validate(config *Config) error {
	v.errors = make([]ValidationError, 0)

	// Validate General section
	v.validateGeneral(&config.General)

	// Validate Categories
	v.validateCategories(config.Categories)

	// Validate Profiles
	v.validateProfiles(config.Profiles)

	// Validate UI section
	v.validateUI(&config.UI)

	// Validate Backup section
	v.validateBackup(&config.Backup)

	// Validate Export section
	v.validateExport(&config.Export)

	// Return combined errors if any
	if len(v.errors) > 0 {
		return fmt.Errorf("configuration validation failed with %d errors: %v", len(v.errors), v.errors)
	}

	return nil
}

// validateGeneral validates the General configuration section
func (v *ConfigValidator) validateGeneral(general *General) {
	// Validate default category
	if general.DefaultCategory == "" {
		v.addError("general.default_category", general.DefaultCategory, "default category cannot be empty")
	} else if !isValidCategoryName(general.DefaultCategory) {
		v.addError("general.default_category", general.DefaultCategory, "invalid category name format")
	}

	// Validate editor
	if general.Editor != "" && !isValidEditor(general.Editor) {
		v.addError("general.editor", general.Editor, "invalid or potentially unsafe editor")
	}
}

// validateCategories validates the Categories configuration
func (v *ConfigValidator) validateCategories(categories map[string]string) {
	if len(categories) == 0 {
		v.addError("categories", categories, "at least one category must be defined")
		return
	}

	for name, description := range categories {
		// Validate category name
		if !isValidCategoryName(name) {
			v.addError(fmt.Sprintf("categories.%s", name), name, "invalid category name format")
		}

		// Validate description
		if len(description) > 200 {
			v.addError(fmt.Sprintf("categories.%s.description", name), description, "description too long (max 200 characters)")
		}

		// Check for dangerous content in description
		if containsSuspiciousContent(description) {
			v.addError(fmt.Sprintf("categories.%s.description", name), description, "description contains potentially dangerous content")
		}
	}
}

// validateProfiles validates the Profiles configuration
func (v *ConfigValidator) validateProfiles(profiles map[string]Profile) {
	defaultCount := 0

	for name, profile := range profiles {
		// Validate profile name
		if !isValidProfileName(name) {
			v.addError(fmt.Sprintf("profiles.%s", name), name, "invalid profile name format")
		}

		// Validate description
		if len(profile.Description) > 200 {
			v.addError(fmt.Sprintf("profiles.%s.description", name), profile.Description, "description too long (max 200 characters)")
		}

		// Check for dangerous content in description
		if containsSuspiciousContent(profile.Description) {
			v.addError(fmt.Sprintf("profiles.%s.description", name), profile.Description, "description contains potentially dangerous content")
		}

		// Validate categories
		if len(profile.Categories) == 0 {
			v.addError(fmt.Sprintf("profiles.%s.categories", name), profile.Categories, "profile must have at least one category")
		}

		for _, categoryName := range profile.Categories {
			if !isValidCategoryName(categoryName) {
				v.addError(fmt.Sprintf("profiles.%s.categories", name), categoryName, "invalid category name in profile")
			}
		}

		// Count default profiles
		if profile.Default {
			defaultCount++
		}
	}

	// Ensure exactly one default profile
	if defaultCount == 0 {
		v.addError("profiles", profiles, "exactly one profile must be marked as default")
	} else if defaultCount > 1 {
		v.addError("profiles", profiles, "only one profile can be marked as default")
	}
}

// validateUI validates the UI configuration section
func (v *ConfigValidator) validateUI(ui *UI) {
	// Validate color scheme
	validColorSchemes := []string{"auto", "light", "dark", "none"}
	if !contains(validColorSchemes, ui.ColorScheme) {
		v.addError("ui.color_scheme", ui.ColorScheme, "invalid color scheme")
	}

	// Validate page size
	if ui.PageSize < 1 || ui.PageSize > 1000 {
		v.addError("ui.page_size", ui.PageSize, "page size must be between 1 and 1000")
	}

	// Validate key bindings
	for action, key := range ui.KeyBindings {
		if !isValidKeyBinding(key) {
			v.addError(fmt.Sprintf("ui.key_bindings.%s", action), key, "invalid key binding format")
		}
	}
}

// validateBackup validates the Backup configuration section
func (v *ConfigValidator) validateBackup(backup *Backup) {
	// Validate directory path
	if backup.Directory != "" && containsSuspiciousPath(backup.Directory) {
		v.addError("backup.directory", backup.Directory, "potentially unsafe directory path")
	}

	// Validate max backups
	if backup.MaxBackups < 1 || backup.MaxBackups > 100 {
		v.addError("backup.max_backups", backup.MaxBackups, "max backups must be between 1 and 100")
	}

	// Validate retention days
	if backup.RetentionDays < 1 || backup.RetentionDays > 3650 {
		v.addError("backup.retention_days", backup.RetentionDays, "retention days must be between 1 and 3650")
	}

	// Validate compression type
	validCompressionTypes := []string{"none", "gzip"}
	if !contains(validCompressionTypes, backup.CompressionType) {
		v.addError("backup.compression_type", backup.CompressionType, "invalid compression type")
	}
}

// validateExport validates the Export configuration section
func (v *ConfigValidator) validateExport(export *Export) {
	// Validate default format
	if export.DefaultFormat == "" {
		v.addError("export.default_format", export.DefaultFormat, "default format cannot be empty")
	}

	// Validate formats
	for name, format := range export.Formats {
		if !isValidFormatName(name) {
			v.addError(fmt.Sprintf("export.formats.%s", name), name, "invalid format name")
		}

		// Validate extension
		if !strings.HasPrefix(format.Extension, ".") {
			v.addError(fmt.Sprintf("export.formats.%s.extension", name), format.Extension, "extension must start with '.'")
		}

		// Validate template for suspicious content
		if containsSuspiciousTemplate(format.Template) {
			v.addError(fmt.Sprintf("export.formats.%s.template", name), format.Template, "template contains potentially dangerous content")
		}
	}
}

// Helper functions

func (v *ConfigValidator) addError(field string, value interface{}, message string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
}

func isValidCategoryName(name string) bool {
	if len(name) == 0 || len(name) > 50 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched
}

func isValidProfileName(name string) bool {
	if len(name) == 0 || len(name) > 50 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched
}

func isValidFormatName(name string) bool {
	if len(name) == 0 || len(name) > 20 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched
}

func isValidEditor(editor string) bool {
	// Use the same validation as in commands.go
	allowedEditors := map[string]bool{
		"nano":         true,
		"vim":          true,
		"vi":           true,
		"emacs":        true,
		"code":         true,
		"notepad":      true,
		"notepad++":    true,
		"sublime_text": true,
		"atom":         true,
		"gedit":        true,
		"kate":         true,
	}

	editorCmd := strings.TrimSpace(editor)

	// Check for suspicious characters
	suspiciousChars := []string{";", "&", "|", "`", "$", "&&", "||", "\n", "\r"}
	for _, char := range suspiciousChars {
		if strings.Contains(editorCmd, char) {
			return false
		}
	}

	// Extract base command name
	parts := strings.Fields(editorCmd)
	if len(parts) == 0 {
		return false
	}

	baseName := strings.ToLower(parts[0])
	if strings.HasSuffix(baseName, ".exe") {
		baseName = strings.TrimSuffix(baseName, ".exe")
	}

	return allowedEditors[baseName]
}

func isValidKeyBinding(key string) bool {
	if len(key) == 0 || len(key) > 10 {
		return false
	}
	// Allow single characters and some special keys
	validKeyPattern := `^[a-zA-Z0-9?/\s]$|^(ctrl|alt|shift)\+[a-zA-Z0-9]$|^F[0-9]{1,2}$`
	matched, _ := regexp.MatchString(validKeyPattern, key)
	return matched
}

func containsSuspiciousContent(content string) bool {
	suspiciousPatterns := []string{
		"<script", "javascript:", "data:", "vbscript:",
		"onclick=", "onload=", "onerror=",
		"eval(", "setTimeout(", "setInterval(",
		"document.", "window.", "alert(",
	}

	lowerContent := strings.ToLower(content)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerContent, pattern) {
			return true
		}
	}

	return false
}

func containsSuspiciousPath(path string) bool {
	suspiciousPatterns := []string{
		"..", "/etc/", "/proc/", "/sys/", "/dev/",
		"C:\\Windows\\", "C:\\System32\\", "\\etc\\",
		"\x00", // null byte
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}

func containsSuspiciousTemplate(template string) bool {
	suspiciousPatterns := []string{
		"{{- range", "{{- if", "{{- with",  // Potentially dangerous template constructs
		"{{ . }}", "{{.}}", // Direct object output without escaping
		"exec", "system", "cmd", "shell",   // System execution functions
		"file", "read", "write",            // File operations
	}

	lowerTemplate := strings.ToLower(template)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerTemplate, pattern) {
			return true
		}
	}

	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}