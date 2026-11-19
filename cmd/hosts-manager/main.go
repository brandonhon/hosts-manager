package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/brandonhon/hosts-manager/internal/audit"
	"github.com/brandonhon/hosts-manager/internal/backup"
	"github.com/brandonhon/hosts-manager/internal/config"
	"github.com/brandonhon/hosts-manager/internal/hosts"
	"github.com/brandonhon/hosts-manager/internal/tui"
	"github.com/brandonhon/hosts-manager/pkg/platform"

	"github.com/spf13/cobra"
)

var (
	cfg     *config.Config
	verbose bool
	dryRun  bool
	// version is set via ldflags during build: -X main.version=<version>
	// Defaults to "dev" for local development builds
	version = "dev"
)

func main() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hosts-manager",
		Short: "Cross-platform hosts file manager",
		Long: `hosts-manager manages your hosts file: entries grouped into categories,
profiles that switch categories on and off, and automatic backups.

Run with no arguments for the interactive editor.`,
		Version: version,
		Args:    cobra.NoArgs,
		// A failed command should print its error, not the whole usage text.
		SilenceUsage: true,
		// main prints the error; letting cobra print it too doubled it.
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}

	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", cfg.General.Verbose, "Enable verbose output")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", cfg.General.DryRun, "Show what would be done without making changes")

	cmd.AddCommand(
		listCmd(),
		addCmd(),
		rmCmd(),
		toggleCmd("enable", true),
		toggleCmd("disable", false),
		profileCmd(),
		backupCmd(),
		restoreCmd(),
		importCmd(),
		exportCmd(),
		configCmd(),
	)

	return cmd
}

// parseHosts reads the system hosts file.
func parseHosts() (*hosts.HostsFile, error) {
	hf, err := hosts.NewParser(platform.New().GetHostsFilePath()).Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse hosts file: %w", err)
	}
	return hf, nil
}

// mutate is the one path by which a command changes the hosts file: check
// privileges, parse, apply change, then report (dry run) or save. change
// returns a past-tense description of what it did, e.g. "Added ...".
func mutate(op string, change func(*hosts.HostsFile) (string, error)) error {
	if err := requireHostsWriteAccess(platform.New()); err != nil {
		return err
	}

	hf, err := parseHosts()
	if err != nil {
		return err
	}

	summary, err := change(hf)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Printf("Dry run, not saved: %s\n", summary)
		return nil
	}

	if err := saveHosts(op, summary, hf); err != nil {
		return err
	}
	fmt.Println(summary)
	return nil
}

// saveHosts backs up the current hosts file (if configured), writes hf in its
// place and records the change in the audit log. The TUI saves through it too.
func saveHosts(op, summary string, hf *hosts.HostsFile) error {
	if cfg.General.AutoBackup {
		path, err := backup.NewManager(cfg).CreateBackup()
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		if verbose {
			fmt.Printf("Backup created: %s\n", path)
		}
	}

	p := platform.New()
	err := hf.Write(p.GetHostsFilePath())
	if logErr := audit.Record(filepath.Join(p.GetDataDir(), "audit"), op, summary, err); logErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not record change in audit log: %v\n", logErr)
	}
	if err != nil {
		return fmt.Errorf("failed to write hosts file: %w", err)
	}
	return nil
}

func runTUI() error {
	// The TUI can save, so it needs the same privileges as every other
	// writing command. Checking here rather than on save means a user
	// without them is told before making edits, not after.
	if err := requireHostsWriteAccess(platform.New()); err != nil {
		return err
	}

	hf, err := parseHosts()
	if err != nil {
		return err
	}

	return tui.Run(hf, cfg, dryRun, func(hf *hosts.HostsFile) error {
		return saveHosts("tui_save", "Saved from TUI", hf)
	})
}

func listCmd() *cobra.Command {
	var category string
	var all bool

	cmd := &cobra.Command{
		Use:   "list [query]",
		Short: "List entries, optionally matching a query",
		Long: `List entries grouped by category. A query matches IPs, hostnames and
comments (case-insensitive substring). Disabled entries are hidden unless --all.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hf, err := parseHosts()
			if err != nil {
				return err
			}

			query := ""
			if len(args) == 1 {
				query = args[0]
			}

			found := false
			for _, c := range hf.Categories {
				if category != "" && c.Name != category {
					continue
				}

				var lines []string
				hidden := 0
				for _, e := range c.Entries {
					if !entryMatches(e, query) {
						continue
					}
					if !e.Enabled && !all {
						hidden++
						continue
					}
					lines = append(lines, formatEntry(e))
				}

				// With a query, skip categories with no match at all.
				if query != "" && len(lines) == 0 && hidden == 0 {
					continue
				}
				found = true

				status := ""
				if !c.Enabled {
					status = ", category disabled"
				}
				if hidden > 0 {
					status += fmt.Sprintf(", %d disabled hidden (-a to show)", hidden)
				}
				fmt.Printf("\n[%s] %d entries%s\n", c.Name, len(c.Entries), status)
				for _, l := range lines {
					fmt.Println(l)
				}
			}

			if category != "" && hf.GetCategory(category) == nil {
				return fmt.Errorf("category not found: %s", category)
			}
			if !found {
				fmt.Println("No entries found")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&category, "category", "c", "", "Only show this category")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Include disabled entries")

	return cmd
}

// entryMatches reports whether e's IP, a hostname or its comment contains
// query, ignoring case. An empty query matches everything.
func entryMatches(e hosts.Entry, query string) bool {
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	fields := append([]string{e.IP, e.Comment}, e.Hostnames...)
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

func formatEntry(e hosts.Entry) string {
	status := "✓"
	if !e.Enabled {
		status = "✗"
	}
	line := fmt.Sprintf("  %s %-15s %s", status, e.IP, strings.Join(e.Hostnames, " "))
	if e.Comment != "" {
		line += "  # " + e.Comment
	}
	return line
}

func addCmd() *cobra.Command {
	var category, comment string

	cmd := &cobra.Command{
		Use:   "add <ip> <hostname> [hostname...]",
		Short: "Add an entry (the category is created if needed)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if category == "" {
				category = cfg.General.DefaultCategory
			}
			entry := hosts.Entry{
				IP:        args[0],
				Hostnames: args[1:],
				Comment:   comment,
				Category:  category,
				Enabled:   true,
			}

			return mutate("add", func(hf *hosts.HostsFile) (string, error) {
				if err := hf.AddEntry(entry); err != nil {
					return "", fmt.Errorf("failed to add entry: %w", err)
				}
				return fmt.Sprintf("Added %s -> %s [%s]", entry.IP, strings.Join(entry.Hostnames, " "), category), nil
			})
		},
	}

	cmd.Flags().StringVarP(&category, "category", "c", "", "Category for the entry")
	cmd.Flags().StringVarP(&comment, "comment", "m", "", "Comment for the entry")

	return cmd
}

func rmCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "rm <hostname>",
		Aliases: []string{"delete"},
		Short:   "Remove a hostname",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hostname := args[0]
			return mutate("delete", func(hf *hosts.HostsFile) (string, error) {
				if !hf.RemoveEntry(hostname) {
					return "", fmt.Errorf("hostname not found: %s", hostname)
				}
				return "Removed " + hostname, nil
			})
		},
	}
}

// toggleCmd builds enable and disable, which act on one hostname or, with
// --category, on a whole category.
func toggleCmd(action string, enable bool) *cobra.Command {
	var category string
	verb := strings.ToUpper(action[:1]) + action[1:] + "d"

	cmd := &cobra.Command{
		Use:   action + " <hostname> | " + action + " -c <category>",
		Short: strings.ToUpper(action[:1]) + action[1:] + " a hostname or a whole category",
		RunE: func(cmd *cobra.Command, args []string) error {
			if category != "" {
				if len(args) != 0 {
					return fmt.Errorf("give a hostname or --category, not both")
				}
				return mutate(action, func(hf *hosts.HostsFile) (string, error) {
					if hf.GetCategory(category) == nil {
						return "", fmt.Errorf("category not found: %s", category)
					}
					if enable {
						hf.EnableCategory(category)
					} else {
						hf.DisableCategory(category)
					}
					return fmt.Sprintf("%s category %s", verb, category), nil
				})
			}

			if len(args) != 1 {
				return fmt.Errorf("requires a hostname, or --category")
			}
			hostname := args[0]
			return mutate(action, func(hf *hosts.HostsFile) (string, error) {
				var ok bool
				if enable {
					ok = hf.EnableEntry(hostname)
				} else {
					ok = hf.DisableEntry(hostname)
				}
				if !ok {
					return "", fmt.Errorf("hostname not found: %s", hostname)
				}
				return fmt.Sprintf("%s %s", verb, hostname), nil
			})
		},
	}

	cmd.Flags().StringVarP(&category, "category", "c", "", "Act on this whole category")

	return cmd
}
