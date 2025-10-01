package main

import (
	"fmt"
	"os"
	"strings"

	"hosts-manager/internal/audit"
	"hosts-manager/internal/backup"
	"hosts-manager/internal/config"
	"hosts-manager/internal/errors"
	"hosts-manager/internal/hosts"
	"hosts-manager/pkg/platform"
	"hosts-manager/pkg/search"

	"github.com/spf13/cobra"
)

var (
	cfg     *config.Config
	verbose bool
	dryRun  bool
	version = "dev" // Will be overridden by ldflags during build
)

func main() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize secure directories at startup to ensure they exist
	// This prevents runtime errors and provides clear user guidance
	if err := ensureSecureDirectories(); err != nil {
		// Log the initialization failure
		if logger, logErr := audit.NewLogger(); logErr == nil {
			logger.LogSecurityViolation("startup", "directory_initialization", err.Error(), nil)
		}
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize secure directories: %v\n", err)
		// Continue execution as some operations might still work
	}

	rootCmd := &cobra.Command{
		Use:   "hosts-manager",
		Short: "Cross-platform hosts file manager",
		Long: `hosts-manager is a cross-platform CLI tool for managing your hosts file.
It provides a template system, backup/restore, interactive TUI mode, and more.`,
		Version: version,
	}
	// Ensure proper initialization and configuration validation

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", cfg.General.Verbose, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", cfg.General.DryRun, "Show what would be done without making changes")

	rootCmd.AddCommand(
		addCmd(),
		listCmd(),
		deleteCmd(),
		enableCmd(),
		disableCmd(),
		searchCmd(),
		backupCmd(),
		restoreCmd(),
		tuiCmd(),
		configCmd(),
		exportCmd(),
		importCmd(),
		categoryCmd(),
		profileCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		// Sanitize error for user display while logging full error for debugging
		if logger, logErr := audit.NewLogger(); logErr == nil && errors.IsSecuritySensitive(err) {
			logger.LogSecurityViolation("command_execution", "root_command", err.Error(), nil)
		}

		sanitizedErr := errors.SanitizeError(err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", sanitizedErr)
		os.Exit(1)
	}
}

func addCmd() *cobra.Command {
	var category, comment string

	cmd := &cobra.Command{
		Use:   "add <ip> <hostname> [hostname...]",
		Short: "Add a new hosts entry",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if category == "" {
				category = cfg.General.DefaultCategory
			}

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

			entry := hosts.Entry{
				IP:        args[0],
				Hostnames: args[1:],
				Comment:   comment,
				Category:  category,
				Enabled:   true,
			}

			if err := hostsFile.AddEntry(entry); err != nil {
				return fmt.Errorf("failed to add entry: %w", err)
			}

			if dryRun {
				fmt.Printf("Would add: %s %s", entry.IP, entry.Hostnames)
				if entry.Comment != "" {
					fmt.Printf(" # %s", entry.Comment)
				}
				fmt.Println()
				return nil
			}

			if err := hostsFile.Write(p.GetHostsFilePath()); err != nil {
				// Log failed operation
				if logger, logErr := audit.NewLogger(); logErr == nil {
					logger.LogHostsOperation("add", entry.IP, entry.Hostnames, false, err.Error())
				}
				return fmt.Errorf("failed to write hosts file: %w", err)
			}

			// Log successful operation
			if logger, err := audit.NewLogger(); err == nil {
				logger.LogHostsOperation("add", entry.IP, entry.Hostnames, true, "")
			}

			fmt.Printf("Added entry: %s -> %v\n", entry.IP, entry.Hostnames)
			return nil
		},
	}

	cmd.Flags().StringVarP(&category, "category", "c", "", "Category for the entry")
	cmd.Flags().StringVar(&comment, "comment", "", "Comment for the entry")

	return cmd
}

func listCmd() *cobra.Command {
	var categoryFilter string
	var showDisabled bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all hosts entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := platform.New()
			parser := hosts.NewParser(p.GetHostsFilePath())
			hostsFile, err := parser.Parse()
			if err != nil {
				return fmt.Errorf("failed to parse hosts file: %w", err)
			}

			for _, category := range hostsFile.Categories {
				if categoryFilter != "" && category.Name != categoryFilter {
					continue
				}

				fmt.Printf("\n=== %s ===\n", category.Name)
				if category.Description != "" {
					fmt.Printf("Description: %s\n", category.Description)
				}
				fmt.Printf("Status: ")
				if category.Enabled {
					fmt.Println("Enabled")
				} else {
					fmt.Println("Disabled")
				}

				for _, entry := range category.Entries {
					if !entry.Enabled && !showDisabled {
						continue
					}

					status := "✓"
					if !entry.Enabled {
						status = "✗"
					}

					fmt.Printf("  %s %s -> %v", status, entry.IP, entry.Hostnames)
					if entry.Comment != "" {
						fmt.Printf(" # %s", entry.Comment)
					}
					fmt.Println()
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&categoryFilter, "category", "c", "", "Filter by category")
	cmd.Flags().BoolVar(&showDisabled, "show-disabled", false, "Show disabled entries")

	return cmd
}

func deleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <hostname>",
		Short: "Delete a hosts entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			hostname := args[0]
			if dryRun {
				fmt.Printf("Would delete hostname: %s\n", hostname)
				return nil
			}

			if !hostsFile.RemoveEntry(hostname) {
				return fmt.Errorf("hostname not found: %s", hostname)
			}

			if err := hostsFile.Write(p.GetHostsFilePath()); err != nil {
				return fmt.Errorf("failed to write hosts file: %w", err)
			}

			fmt.Printf("Deleted hostname: %s\n", hostname)
			return nil
		},
	}

	return cmd
}

func enableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <hostname>",
		Short: "Enable a hosts entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return toggleEntry(args[0], true)
		},
	}

	return cmd
}

func disableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <hostname>",
		Short: "Disable a hosts entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return toggleEntry(args[0], false)
		},
	}

	return cmd
}

func toggleEntry(hostname string, enable bool) error {
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
		fmt.Printf("Would %s hostname: %s\n", action, hostname)
		return nil
	}

	var success bool
	if enable {
		success = hostsFile.EnableEntry(hostname)
	} else {
		success = hostsFile.DisableEntry(hostname)
	}

	if !success {
		return fmt.Errorf("hostname not found: %s", hostname)
	}

	if err := hostsFile.Write(p.GetHostsFilePath()); err != nil {
		return fmt.Errorf("failed to write hosts file: %w", err)
	}

	// Capitalize first letter manually (strings.Title is deprecated)
	actionCapitalized := strings.ToUpper(action[:1]) + action[1:]
	fmt.Printf("%sd hostname: %s\n", actionCapitalized, hostname)
	return nil
}

func searchCmd() *cobra.Command {
	var fuzzy bool
	var caseSensitive bool
	var categoryFilter string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search hosts entries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := platform.New()
			parser := hosts.NewParser(p.GetHostsFilePath())
			hostsFile, err := parser.Parse()
			if err != nil {
				return fmt.Errorf("failed to parse hosts file: %w", err)
			}

			searcher := search.NewSearcher(caseSensitive, fuzzy)
			var results []search.Result

			if categoryFilter != "" {
				results = searcher.SearchByCategory(hostsFile, args[0], categoryFilter)
			} else {
				results = searcher.Search(hostsFile, args[0])
			}

			if len(results) == 0 {
				fmt.Println("No entries found")
				return nil
			}

			fmt.Printf("Found %d entries:\n\n", len(results))
			for _, result := range results {
				entry := result.Entry
				status := "✓"
				if !entry.Enabled {
					status = "✗"
				}

				fmt.Printf("  %s [%s] %s -> %v (score: %.2f, match: %s)",
					status, entry.Category, entry.IP, entry.Hostnames, result.Score, result.Match)
				if entry.Comment != "" {
					fmt.Printf(" # %s", entry.Comment)
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&fuzzy, "fuzzy", true, "Enable fuzzy matching")
	cmd.Flags().BoolVar(&caseSensitive, "case-sensitive", false, "Enable case-sensitive search")
	cmd.Flags().StringVarP(&categoryFilter, "category", "c", "", "Filter by category")

	return cmd
}
