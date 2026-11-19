package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/brandonhon/hosts-manager/internal/backup"
	"github.com/brandonhon/hosts-manager/internal/config"
	"github.com/brandonhon/hosts-manager/internal/hosts"
	"github.com/brandonhon/hosts-manager/internal/platform"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// requireHostsWriteAccess verifies the privileges needed to modify the hosts
// file, unless this is a dry run.
//
// A dry run writes nothing, so demanding root to preview a change was pure
// friction — and it made `--dry-run` useless to anyone without sudo, which is
// the one audience a preview is for.
func requireHostsWriteAccess(p *platform.Platform) error {
	if dryRun {
		return nil
	}
	return p.ElevateIfNeeded()
}

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
	return &cobra.Command{
		Use:   "restore [number|file]",
		Short: "List backups, or restore one",
		Long: `With no argument, list backups newest first. Restore one by its number
in that list (1 is the newest) or by file name.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupMgr := backup.NewManager(cfg)
			backups, err := backupMgr.ListBackups()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				if len(backups) == 0 {
					fmt.Println("No backups found")
					return nil
				}
				for i, b := range backups {
					fmt.Printf("%2d. %s  %s  %s\n", i+1,
						b.Timestamp.Format("2006-01-02 15:04:05"),
						formatSize(b.Size), filepath.Base(b.FilePath))
				}
				return nil
			}

			if err := requireHostsWriteAccess(platform.New()); err != nil {
				return err
			}

			target := args[0]
			if n, err := strconv.Atoi(target); err == nil {
				if n < 1 || n > len(backups) {
					return fmt.Errorf("no backup numbered %d; run 'restore' to list them", n)
				}
				target = backups[n-1].FilePath
			}

			backupPath, err := validateFilePath(target, cfg.Backup.Directory)
			if err != nil {
				return fmt.Errorf("invalid backup path: %w", err)
			}

			if dryRun {
				fmt.Printf("Dry run, not saved: Restored %s\n", filepath.Base(backupPath))
				return nil
			}
			if err := backupMgr.RestoreBackup(backupPath); err != nil {
				return err
			}
			fmt.Printf("Restored %s\n", filepath.Base(backupPath))
			return nil
		},
	}
}

func configCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show the config file path and current settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			fmt.Printf("# %s\n%s", filepath.Join(platform.New().GetConfigDir(), "config.yaml"), data)
			return nil
		},
	}
}

func exportCmd() *cobra.Command {
	var format string
	var output string
	var categoryFilter string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export hosts entries",
		Long:  `Export entries as json, yaml or hosts format, to stdout or a file (-o).`,
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
				return usageErrorf(cmd, "unsupported format %q: use json, yaml or hosts", format)
			}

			if err != nil {
				return err
			}

			if output == "" {
				fmt.Print(string(data))
			} else {
				if err := os.WriteFile(output, data, 0600); err != nil {
					return err
				}
				if err := platform.GiveToInvokingUser(output); err != nil {
					return err
				}
				fmt.Printf("Exported to: %s\n", output)
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
		Long: `Import entries from a json or yaml file (as written by export). Replaces
all entries unless --merge. Every entry is validated before anything is written.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Reading a file the user named is the point of import, and its
			// contents are validated before anything reaches the hosts file.
			// #nosec G304 -- user-chosen input file, contents validated below
			data, err := os.ReadFile(args[0])
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
				return usageErrorf(cmd, "unsupported import format %q: use json or yaml", format)
			}

			if err != nil {
				return fmt.Errorf("failed to parse import file: %w", err)
			}

			// A file containing literal "null" unmarshals to a nil pointer
			// without erroring.
			if importedHosts == nil {
				return fmt.Errorf("import file contains no hosts data")
			}

			// Validate before writing anything. The merge path below reaches
			// AddEntry, which validates, but the replace path went straight to
			// Write -- so an import file could carry values no other code path
			// would accept, including a comment with an embedded newline that
			// re-parses as an extra host mapping. Reject the whole file rather
			// than importing it partially.
			if err := validateImportedHosts(importedHosts); err != nil {
				return err
			}

			return mutate("import", func(hf *hosts.HostsFile) (string, error) {
				if !merge {
					// Replace everything except where it is written.
					path := hf.FilePath
					*hf = *importedHosts
					hf.FilePath = path
					return fmt.Sprintf("Imported %d categories (replaced existing entries)", len(hf.Categories)), nil
				}
				n := 0
				for _, category := range importedHosts.Categories {
					for _, entry := range category.Entries {
						if err := hf.AddEntry(entry); err != nil {
							return "", fmt.Errorf("failed to add imported entry %s: %w", entry.IP, err)
						}
						n++
					}
				}
				return fmt.Sprintf("Merged %d imported entries", n), nil
			})
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "yaml", "Import format (json, yaml)")
	cmd.Flags().BoolVarP(&merge, "merge", "m", false, "Merge with existing entries")

	return cmd
}

func profileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profile [name]",
		Short: "List profiles, or activate one",
		Long: `With no argument, list profiles (* marks the active one). With a name,
enable exactly that profile's categories and disable the rest.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				names := make([]string, 0, len(cfg.Profiles))
				for name := range cfg.Profiles {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					prof := cfg.Profiles[name]
					mark := " "
					if prof.Default {
						mark = "*"
					}
					fmt.Printf("%s %-12s %s %v\n", mark, name, prof.Description, prof.Categories)
				}
				return nil
			}

			name := args[0]
			profile, ok := cfg.Profiles[name]
			if !ok {
				return fmt.Errorf("profile not found: %s", name)
			}

			err := mutate("profile_activate", func(hf *hosts.HostsFile) (string, error) {
				hf.SetEnabledCategories(profile.Categories)
				return fmt.Sprintf("Activated profile %s %v", name, profile.Categories), nil
			})
			if err != nil || dryRun {
				return err
			}

			for n, prof := range cfg.Profiles {
				prof.Default = n == name
				cfg.Profiles[n] = prof
			}
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			return nil
		},
	}
}

// validateImportedHosts runs every entry in an imported file through the same
// validation an interactively added entry gets. Category names are checked too,
// since an imported category name reaches the hosts file as an @category marker.
func validateImportedHosts(hostsFile *hosts.HostsFile) error {
	for _, category := range hostsFile.Categories {
		for _, entry := range category.Entries {
			// An entry carries its own category, but the enclosing category is
			// what the writer actually groups by, so check that name as well.
			candidate := entry
			if candidate.Category == "" {
				candidate.Category = category.Name
			}
			if err := hosts.ValidateEntry(candidate); err != nil {
				return fmt.Errorf("invalid imported entry %s (%s): %w",
					candidate.IP, strings.Join(candidate.Hostnames, " "), err)
			}
		}
	}
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

		// The @category marker is what the parser reads back; the banner below
		// it is decoration. Without the marker an exported hosts file re-parses
		// with every entry in the default category.
		categoryHeader := fmt.Sprintf("# @category %s", category.Name)
		if category.Description != "" {
			categoryHeader += " " + category.Description
		}
		builder.WriteString(categoryHeader + "\n")

		fmt.Fprintf(&builder, "# =============== %s ===============\n", strings.ToUpper(category.Name))

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
