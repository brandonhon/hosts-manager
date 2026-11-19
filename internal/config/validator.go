package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
)

// nameRegex matches category and profile names. A category name is written to
// the hosts file as an @category marker, so this also keeps it to one token.
var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,50}$`)

// Validate reports every problem with config at once, so a user fixing their
// config file sees the whole list rather than one error per attempt.
func Validate(config *Config) error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	if !nameRegex.MatchString(config.General.DefaultCategory) {
		add("general.default_category %q: must be 1-50 letters, digits, '-' or '_'", config.General.DefaultCategory)
	}

	defaults := 0
	for name, profile := range config.Profiles {
		if !nameRegex.MatchString(name) {
			add("profiles.%s: name must be 1-50 letters, digits, '-' or '_'", name)
		}
		if len(profile.Categories) == 0 {
			add("profiles.%s: must list at least one category", name)
		}
		for _, c := range profile.Categories {
			if !nameRegex.MatchString(c) {
				add("profiles.%s: invalid category name %q", name, c)
			}
		}
		if profile.Default {
			defaults++
		}
	}
	if len(config.Profiles) > 0 && defaults != 1 {
		add("profiles: exactly one must be marked default, found %d", defaults)
	}

	if config.Backup.MaxBackups < 1 || config.Backup.MaxBackups > 100 {
		add("backup.max_backups %d: must be between 1 and 100", config.Backup.MaxBackups)
	}
	if config.Backup.RetentionDays < 1 || config.Backup.RetentionDays > 3650 {
		add("backup.retention_days %d: must be between 1 and 3650", config.Backup.RetentionDays)
	}
	if !slices.Contains([]string{"none", "gzip"}, config.Backup.CompressionType) {
		add("backup.compression_type %q: must be none or gzip", config.Backup.CompressionType)
	}

	if !slices.Contains([]string{"json", "yaml", "hosts"}, config.Export.DefaultFormat) {
		add("export.default_format %q: must be json, yaml or hosts", config.Export.DefaultFormat)
	}

	return errors.Join(errs...)
}
