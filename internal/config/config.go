package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/brandonhon/hosts-manager/internal/platform"

	"gopkg.in/yaml.v3"
)

type Config struct {
	General  General            `yaml:"general"`
	Profiles map[string]Profile `yaml:"profiles"`
	Backup   Backup             `yaml:"backup"`
	Export   Export             `yaml:"export"`
}

type General struct {
	DefaultCategory string `yaml:"default_category"`
	AutoBackup      bool   `yaml:"auto_backup"`
	DryRun          bool   `yaml:"dry_run"`
	Verbose         bool   `yaml:"verbose"`
}

type Profile struct {
	Description string   `yaml:"description"`
	Categories  []string `yaml:"categories"`
	Default     bool     `yaml:"default"`
}

type Backup struct {
	Directory       string `yaml:"directory"`
	MaxBackups      int    `yaml:"max_backups"`
	RetentionDays   int    `yaml:"retention_days"`
	CompressionType string `yaml:"compression_type"`
}

type Export struct {
	DefaultFormat string `yaml:"default_format"`
}

func DefaultConfig() *Config {
	return &Config{
		General: General{
			DefaultCategory: "custom",
			AutoBackup:      true,
			DryRun:          false,
			Verbose:         false,
		},
		Profiles: map[string]Profile{
			"minimal": {
				Description: "Minimal profile with essential entries only",
				Categories:  []string{"production"},
				Default:     false,
			},
			"development": {
				Description: "Development profile with dev and staging environments",
				Categories:  []string{"development", "staging"},
				Default:     false,
			},
			"full": {
				Description: "Full profile with all categories enabled",
				Categories:  []string{"development", "staging", "production", "custom"},
				Default:     true,
			},
		},
		Backup: Backup{
			Directory:       "",
			MaxBackups:      10,
			RetentionDays:   30,
			CompressionType: "gzip",
		},
		Export: Export{
			DefaultFormat: "yaml",
		},
	}
}

func Load() (*Config, error) {
	p := platform.New()
	configDir := p.GetConfigDir()
	configPath := filepath.Join(configDir, "config.yaml")

	// Both paths converge below. They used to be separate, and only the
	// read path filled in the backup directory — so the very first run
	// returned a config with an empty one and every command with AutoBackup
	// enabled (the default) failed at os.MkdirAll("").
	config := DefaultConfig()
	firstRun := false

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		firstRun = true
	} else {
		// configPath is filepath.Join(p.GetConfigDir(), "config.yaml") — built
		// by this function from the platform config dir, not supplied.
		// #nosec G304 -- fixed filename under the platform config directory
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Written before the backup directory is resolved, so the file on disk
	// keeps an empty value. An empty directory means "work it out from the
	// data dir at load time", which survives the home directory moving; a
	// resolved absolute path baked into the file would not.
	if firstRun {
		if err := Save(config); err != nil {
			return config, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	if config.Backup.Directory == "" {
		config.Backup.Directory = filepath.Join(p.GetDataDir(), "backups")
	}

	if err := Validate(config); err != nil {
		return nil, fmt.Errorf("invalid config %s:\n%w", configPath, err)
	}

	return config, nil
}

func Save(config *Config) error {
	if err := Validate(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	p := platform.New()
	configDir := p.GetConfigDir()

	if err := platform.MkdirAllForUser(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	if err := platform.GiveToInvokingUser(configPath); err != nil {
		return fmt.Errorf("failed to set config file owner: %w", err)
	}

	return nil
}
