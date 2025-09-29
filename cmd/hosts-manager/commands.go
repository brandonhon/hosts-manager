package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"hosts-manager/internal/audit"
	"hosts-manager/internal/backup"
	"hosts-manager/internal/config"
	"hosts-manager/internal/hosts"
	"hosts-manager/internal/tui"
	"hosts-manager/pkg/platform"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func backupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create a backup of the hosts file",
		RunE: func(cmd *cobra.Command, args []string) error {
			backupMgr := backup.NewManager(cfg)
			backupPath, err := backupMgr.CreateBackup()
			if err != nil {
				return err
			}

			fmt.Printf("Backup created: %s\n", backupPath)
			return nil
		},
	}

	return cmd
}

func restoreCmd() *cobra.Command {
	var listBackups bool

	cmd := &cobra.Command{
		Use:   "restore [backup-file]",
		Short: "Restore hosts file from backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			backupMgr := backup.NewManager(cfg)

			if listBackups {
				backups, err := backupMgr.ListBackups()
				if err != nil {
					return err
				}

				if len(backups) == 0 {
					fmt.Println("No backups found")
					return nil
				}

				fmt.Println("Available backups:")
				for i, backup := range backups {
					fmt.Printf("%d. %s (%s, %s)\n",
						i+1,
						filepath.Base(backup.FilePath),
						backup.Timestamp.Format("2006-01-02 15:04:05"),
						formatSize(backup.Size))
				}
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("backup file path required. Use --list to see available backups")
			}

			p := platform.New()
			if err := p.ElevateIfNeeded(); err != nil {
				return err
			}

			userPath := args[0]

			// Validate and secure the backup path
			backupPath, err := validateFilePath(userPath, cfg.Backup.Directory)
			if err != nil {
				return fmt.Errorf("invalid backup path: %w", err)
			}

			return backupMgr.RestoreBackup(backupPath)
		},
	}

	cmd.Flags().BoolVarP(&listBackups, "list", "l", false, "List available backups")

	return cmd
}

func tuiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Start interactive TUI mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := platform.New()
			parser := hosts.NewParser(p.GetHostsFilePath())
			hostsFile, err := parser.Parse()
			if err != nil {
				return fmt.Errorf("failed to parse hosts file: %w", err)
			}

			return tui.Run(hostsFile, cfg)
		},
	}

	return cmd
}

func configCmd() *cobra.Command {
	var show bool
	var edit bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if show {
				data, err := yaml.Marshal(cfg)
				if err != nil {
					return err
				}
				fmt.Print(string(data))
				return nil
			}

			if edit {
				p := platform.New()
				configPath := filepath.Join(p.GetConfigDir(), "config.yaml")
				editor := cfg.General.Editor

				if editor == "" {
					editor = "nano"
				}

				// Validate editor command for security
				if !isValidEditor(editor) {
					return fmt.Errorf("editor '%s' is not allowed for security reasons. Allowed editors: nano, vim, vi, emacs, code, notepad", editor)
				}

				return runCommand(editor, configPath)
			}

			return cmd.Help()
		},
	}

	cmd.Flags().BoolVar(&show, "show", false, "Show current configuration")
	cmd.Flags().BoolVar(&edit, "edit", false, "Edit configuration file")

	return cmd
}

func exportCmd() *cobra.Command {
	var format string
	var output string
	var categoryFilter string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export hosts entries",
		Long: `Export hosts file to different format (json, yaml, hosts).

For security, export operations are restricted to these directories:
• ~/.local/share/hosts-manager/ (data directory)
• ~/.config/hosts-manager/ (config directory)
• /tmp/hosts-manager/ (temporary directory)

Use relative paths (e.g., 'my-export.json') or paths within these directories.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := platform.New()
			parser := hosts.NewParser(p.GetHostsFilePath())
			hostsFile, err := parser.Parse()
			if err != nil {
				return fmt.Errorf("failed to parse hosts file: %w", err)
			}

			if categoryFilter != "" {
				filteredCategories := []hosts.Category{}
				for _, category := range hostsFile.Categories {
					if category.Name == categoryFilter {
						filteredCategories = append(filteredCategories, category)
						break
					}
				}
				hostsFile.Categories = filteredCategories
			}

			var data []byte
			switch format {
			case "json":
				data, err = json.MarshalIndent(hostsFile, "", "  ")
			case "yaml":
				data, err = yaml.Marshal(hostsFile)
			case "hosts":
				data, err = exportToHosts(hostsFile)
			default:
				return fmt.Errorf("unsupported format: %s", format)
			}

			if err != nil {
				return err
			}

			if output == "" {
				fmt.Print(string(data))
			} else {
				// Ensure secure directories exist
				if err := ensureSecureDirectories(); err != nil {
					return fmt.Errorf("failed to initialize secure directories: %w", err)
				}

				// Validate output path using secure directory restrictions
				allowedDirs := getAllowedDirectories()
				outputPath, err := validateFilePathStrict(output, allowedDirs, "export")
				if err != nil {
					return fmt.Errorf("export path validation failed: %w", err)
				}

				if err := os.WriteFile(outputPath, data, 0600); err != nil {
					return err
				}
				fmt.Printf("Exported to: %s\n", outputPath)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", cfg.Export.DefaultFormat, "Export format (json, yaml, hosts)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")
	cmd.Flags().StringVarP(&categoryFilter, "category", "c", "", "Export only specific category")

	return cmd
}

func importCmd() *cobra.Command {
	var format string
	var merge bool

	cmd := &cobra.Command{
		Use:   "import <file>",
		Short: "Import hosts entries from file",
		Long: `Import hosts entries from a file (json or yaml format).

For security, import operations are restricted to these directories:
• ~/.local/share/hosts-manager/ (data directory)
• ~/.config/hosts-manager/ (config directory)
• /tmp/hosts-manager/ (temporary directory)

Use relative paths (e.g., 'my-import.json') or paths within these directories.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := platform.New()
			if err := p.ElevateIfNeeded(); err != nil {
				return err
			}

			userPath := args[0]

			// Ensure secure directories exist
			if err := ensureSecureDirectories(); err != nil {
				return fmt.Errorf("failed to initialize secure directories: %w", err)
			}

			// Validate import file path using secure directory restrictions
			allowedDirs := getAllowedDirectories()
			filePath, err := validateFilePathStrict(userPath, allowedDirs, "import")
			if err != nil {
				return fmt.Errorf("import path validation failed: %w", err)
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read import file: %w", err)
			}

			var importedHosts *hosts.HostsFile
			switch format {
			case "json":
				err = json.Unmarshal(data, &importedHosts)
			case "yaml":
				err = yaml.Unmarshal(data, &importedHosts)
			default:
				return fmt.Errorf("unsupported import format: %s", format)
			}

			if err != nil {
				return fmt.Errorf("failed to parse import file: %w", err)
			}

			if merge {
				parser := hosts.NewParser(p.GetHostsFilePath())
				currentHosts, err := parser.Parse()
				if err != nil {
					return fmt.Errorf("failed to parse current hosts file: %w", err)
				}

				for _, category := range importedHosts.Categories {
					for _, entry := range category.Entries {
						if err := currentHosts.AddEntry(entry); err != nil {
							return fmt.Errorf("failed to add imported entry %s: %w", entry.IP, err)
						}
					}
				}
				importedHosts = currentHosts
			}

			backupMgr := backup.NewManager(cfg)
			if cfg.General.AutoBackup {
				if _, err := backupMgr.CreateBackup(); err != nil {
					return fmt.Errorf("failed to create backup: %w", err)
				}
				if verbose {
					fmt.Println("Backup created successfully")
				}
			}

			if dryRun {
				fmt.Printf("Would import %d categories with entries\n", len(importedHosts.Categories))
				for _, category := range importedHosts.Categories {
					fmt.Printf("  %s: %d entries\n", category.Name, len(category.Entries))
				}
				return nil
			}

			if err := importedHosts.Write(p.GetHostsFilePath()); err != nil {
				return fmt.Errorf("failed to write hosts file: %w", err)
			}

			fmt.Printf("Successfully imported %d categories\n", len(importedHosts.Categories))
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "yaml", "Import format (json, yaml)")
	cmd.Flags().BoolVarP(&merge, "merge", "m", false, "Merge with existing entries")

	return cmd
}

func categoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "category",
		Short: "Manage categories",
	}

	cmd.AddCommand(categoryListCmd())
	cmd.AddCommand(categoryEnableCmd())
	cmd.AddCommand(categoryDisableCmd())

	return cmd
}

func categoryListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all categories",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := platform.New()
			parser := hosts.NewParser(p.GetHostsFilePath())
			hostsFile, err := parser.Parse()
			if err != nil {
				return fmt.Errorf("failed to parse hosts file: %w", err)
			}

			fmt.Println("Categories:")
			for _, category := range hostsFile.Categories {
				status := "✓"
				if !category.Enabled {
					status = "✗"
				}

				fmt.Printf("  %s %s (%d entries)", status, category.Name, len(category.Entries))
				if category.Description != "" {
					fmt.Printf(" - %s", category.Description)
				}
				fmt.Println()
			}

			return nil
		},
	}

	return cmd
}

func categoryEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <category>",
		Short: "Enable a category and all its entries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return toggleCategory(args[0], true)
		},
	}

	return cmd
}

func categoryDisableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <category>",
		Short: "Disable a category and all its entries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return toggleCategory(args[0], false)
		},
	}

	return cmd
}

func profileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage profiles",
	}

	cmd.AddCommand(profileListCmd())
	cmd.AddCommand(profileActivateCmd())

	return cmd
}

func profileListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Available profiles:")
			for name, profile := range cfg.Profiles {
				status := " "
				if profile.Default {
					status = "*"
				}

				fmt.Printf("  %s %s - %s\n", status, name, profile.Description)
				fmt.Printf("    Categories: %v\n", profile.Categories)
			}

			return nil
		},
	}

	return cmd
}

func profileActivateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activate <profile>",
		Short: "Activate a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := args[0]
			profile, exists := cfg.Profiles[profileName]
			if !exists {
				return fmt.Errorf("profile not found: %s", profileName)
			}

			p := platform.New()
			if err := p.ElevateIfNeeded(); err != nil {
				return err
			}

			parser := hosts.NewParser(p.GetHostsFilePath())
			hostsFile, err := parser.Parse()
			if err != nil {
				return fmt.Errorf("failed to parse hosts file: %w", err)
			}

			backupMgr := backup.NewManager(cfg)
			if cfg.General.AutoBackup {
				if _, err := backupMgr.CreateBackup(); err != nil {
					return fmt.Errorf("failed to create backup: %w", err)
				}
				if verbose {
					fmt.Println("Backup created successfully")
				}
			}

			for i := range hostsFile.Categories {
				category := &hostsFile.Categories[i]
				enabled := false
				for _, activeCat := range profile.Categories {
					if category.Name == activeCat {
						enabled = true
						break
					}
				}

				category.Enabled = enabled
				for j := range category.Entries {
					category.Entries[j].Enabled = enabled
				}
			}

			if dryRun {
				fmt.Printf("Would activate profile: %s\n", profileName)
				fmt.Printf("Enabled categories: %v\n", profile.Categories)
				return nil
			}

			if err := hostsFile.Write(p.GetHostsFilePath()); err != nil {
				return fmt.Errorf("failed to write hosts file: %w", err)
			}

			for name, prof := range cfg.Profiles {
				prof.Default = (name == profileName)
				cfg.Profiles[name] = prof
			}

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Activated profile: %s\n", profileName)
			return nil
		},
	}

	return cmd
}

func toggleCategory(categoryName string, enable bool) error {
	p := platform.New()
	if err := p.ElevateIfNeeded(); err != nil {
		return err
	}

	backupMgr := backup.NewManager(cfg)
	if cfg.General.AutoBackup {
		if _, err := backupMgr.CreateBackup(); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		if verbose {
			fmt.Println("Backup created successfully")
		}
	}

	parser := hosts.NewParser(p.GetHostsFilePath())
	hostsFile, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse hosts file: %w", err)
	}

	action := "disable"
	if enable {
		action = "enable"
	}

	if dryRun {
		fmt.Printf("Would %s category: %s\n", action, categoryName)
		return nil
	}

	if enable {
		hostsFile.EnableCategory(categoryName)
	} else {
		hostsFile.DisableCategory(categoryName)
	}

	if err := hostsFile.Write(p.GetHostsFilePath()); err != nil {
		return fmt.Errorf("failed to write hosts file: %w", err)
	}

	// Capitalize first letter manually (strings.Title is deprecated)
	actionCapitalized := strings.ToUpper(action[:1]) + action[1:]
	fmt.Printf("%sd category: %s\n", actionCapitalized, categoryName)
	return nil
}

func exportToHosts(hostsFile *hosts.HostsFile) ([]byte, error) {
	var builder strings.Builder

	for _, headerLine := range hostsFile.Header {
		builder.WriteString(headerLine + "\n")
	}

	if len(hostsFile.Header) > 0 {
		builder.WriteString("\n")
	}

	for _, category := range hostsFile.Categories {
		if !category.Enabled || len(category.Entries) == 0 {
			continue
		}

		builder.WriteString(fmt.Sprintf("# =============== %s ===============\n", strings.ToUpper(category.Name)))

		for _, entry := range category.Entries {
			if !entry.Enabled {
				continue
			}

			line := fmt.Sprintf("%s %s", entry.IP, strings.Join(entry.Hostnames, " "))
			if entry.Comment != "" {
				line += " # " + entry.Comment
			}
			builder.WriteString(line + "\n")
		}

		builder.WriteString("\n")
	}

	for _, footerLine := range hostsFile.Footer {
		builder.WriteString(footerLine + "\n")
	}

	return []byte(builder.String()), nil
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// getAllowedDirectories returns the list of directories where file operations are permitted
func getAllowedDirectories() []string {
	p := platform.New()
	return []string{
		p.GetDataDir(),   // User data directory (e.g., ~/.local/share/hosts-manager)
		p.GetConfigDir(), // User config directory (e.g., ~/.config/hosts-manager)
		filepath.Join(os.TempDir(), "hosts-manager"), // Secure temp directory
		// Note: backup directory is handled separately in cfg.Backup.Directory
	}
}

// ensureSecureDirectories creates the allowed directories with proper permissions
func ensureSecureDirectories() error {
	dirs := getAllowedDirectories()
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create secure directory %s: %w", dir, err)
		}
	}
	return nil
}

// validateFilePathStrict validates file paths with mandatory directory restrictions
func validateFilePathStrict(filePath string, allowedDirs []string, operation string) (string, error) {
	if len(allowedDirs) == 0 {
		return "", fmt.Errorf("no allowed directories specified for %s operation", operation)
	}

	// Log security-sensitive path validation attempts
	if logger, err := audit.NewLogger(); err == nil {
		logger.LogFileOperation(operation+"_path_validation", filePath, true, "")
	}

	// Clean the path to resolve any ".." or similar elements
	cleanPath := filepath.Clean(filePath)

	// Additional security checks first
	if strings.Contains(cleanPath, "\x00") {
		if logger, err := audit.NewLogger(); err == nil {
			logger.LogSecurityViolation("path_validation", filePath, "null byte detected in path", map[string]interface{}{
				"operation": operation,
				"path":      filePath,
			})
		}
		return "", fmt.Errorf("invalid path: contains null byte")
	}

	// Try to validate against each allowed directory
	var validationErrors []string
	for _, allowedDir := range allowedDirs {
		if validatedPath, err := validateFilePath(cleanPath, allowedDir); err == nil {
			return validatedPath, nil
		} else {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: %v", allowedDir, err))
		}
	}

	// If no allowed directory worked, this is a security violation
	if logger, err := audit.NewLogger(); err == nil {
		logger.LogSecurityViolation("path_validation", filePath, "path not in allowed directories", map[string]interface{}{
			"operation":         operation,
			"path":              filePath,
			"allowed_dirs":      allowedDirs,
			"validation_errors": validationErrors,
		})
	}

	return "", fmt.Errorf("%s operation denied: path '%s' is not within allowed directories: %v",
		operation, filePath, allowedDirs)
}

// validateFilePath validates that a file path is safe and prevents path traversal attacks
// This function now requires an allowedDir to be specified for security
func validateFilePath(filePath string, allowedDir string) (string, error) {
	if allowedDir == "" {
		return "", fmt.Errorf("security error: allowed directory must be specified for path validation")
	}

	// Clean the path to resolve any ".." or similar elements
	cleanPath := filepath.Clean(filePath)

	// Convert to absolute path
	var absPath string
	if filepath.IsAbs(cleanPath) {
		absPath = cleanPath
	} else {
		absPath = filepath.Join(allowedDir, cleanPath)
	}

	// Clean again after joining
	absPath = filepath.Clean(absPath)

	// Get absolute path of allowed directory
	allowedDirAbs, err := filepath.Abs(allowedDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve allowed directory: %w", err)
	}

	// Ensure the file path is within the allowed directory
	relPath, err := filepath.Rel(allowedDirAbs, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}

	// Check if the relative path tries to escape the allowed directory
	if strings.HasPrefix(relPath, ".."+string(filepath.Separator)) || relPath == ".." {
		return "", fmt.Errorf("path traversal attempt detected: %s", filePath)
	}

	// Additional security checks
	if strings.Contains(absPath, "\x00") {
		return "", fmt.Errorf("null byte in path")
	}

	return absPath, nil
}

// isValidEditor validates that the editor command is safe to execute
func isValidEditor(editor string) bool {
	// Whitelist of allowed editors - only the base command name, no arguments
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

	// Extract just the command name (no paths, no arguments)
	editorCmd := strings.TrimSpace(editor)

	// Reject if contains suspicious characters
	if strings.Contains(editorCmd, ";") ||
		strings.Contains(editorCmd, "&") ||
		strings.Contains(editorCmd, "|") ||
		strings.Contains(editorCmd, "`") ||
		strings.Contains(editorCmd, "$") ||
		strings.Contains(editorCmd, "&&") ||
		strings.Contains(editorCmd, "||") {
		return false
	}

	// Extract just the base command (handle full paths)
	baseName := filepath.Base(editorCmd)

	// Remove .exe extension on Windows
	baseName = strings.TrimSuffix(baseName, ".exe")

	return allowedEditors[baseName]
}

func runCommand(name string, args ...string) error {
	// Additional security validation before execution
	if strings.ContainsRune(name, 0) {
		if logger, err := audit.NewLogger(); err == nil {
			logger.LogSecurityViolation("command_execution", name, "null byte in command name", map[string]interface{}{
				"command": name,
				"args":    args,
			})
		}
		return fmt.Errorf("invalid command: contains null byte")
	}

	// Validate arguments for suspicious content
	for i, arg := range args {
		if strings.ContainsRune(arg, 0) {
			if logger, err := audit.NewLogger(); err == nil {
				logger.LogSecurityViolation("command_execution", name, "null byte in command argument", map[string]interface{}{
					"command":   name,
					"arg_index": i,
					"arg_value": arg,
				})
			}
			return fmt.Errorf("invalid argument: contains null byte")
		}
	}

	// Log the command execution attempt for audit trail
	if logger, err := audit.NewLogger(); err == nil {
		logger.LogFileOperation("editor_execution", name, true, "")
	}

	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	// Log execution result
	if logger, logErr := audit.NewLogger(); logErr == nil {
		success := err == nil
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}
		logger.LogFileOperation("editor_execution_result", name, success, errorMsg)
	}

	return err
}
