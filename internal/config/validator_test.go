package config

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string // substring; "" means valid
	}{
		{"defaults are valid", func(*Config) {}, ""},
		{"empty default category", func(c *Config) { c.General.DefaultCategory = "" }, "general.default_category"},
		{"default category with space", func(c *Config) { c.General.DefaultCategory = "a b" }, "general.default_category"},
		{"bad profile name", func(c *Config) {
			c.Profiles["bad name"] = Profile{Categories: []string{"x"}}
		}, "profiles.bad name"},
		{"profile without categories", func(c *Config) {
			p := c.Profiles["minimal"]
			p.Categories = nil
			c.Profiles["minimal"] = p
		}, "at least one category"},
		{"two default profiles", func(c *Config) {
			p := c.Profiles["minimal"]
			p.Default = true
			c.Profiles["minimal"] = p
		}, "found 2"},
		{"no profiles is fine", func(c *Config) { c.Profiles = nil }, ""},
		{"max backups zero", func(c *Config) { c.Backup.MaxBackups = 0 }, "backup.max_backups"},
		{"retention too long", func(c *Config) { c.Backup.RetentionDays = 4000 }, "backup.retention_days"},
		{"unknown compression", func(c *Config) { c.Backup.CompressionType = "zip" }, "backup.compression_type"},
		{"unknown export format", func(c *Config) { c.Export.DefaultFormat = "xml" }, "export.default_format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DefaultConfig()
			tt.mutate(c)
			err := Validate(c)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("error = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}

// Every problem is reported, not just the first.
func TestValidateReportsAllErrors(t *testing.T) {
	c := DefaultConfig()
	c.General.DefaultCategory = ""
	c.Backup.MaxBackups = 0
	err := Validate(c)
	if err == nil || !strings.Contains(err.Error(), "default_category") || !strings.Contains(err.Error(), "max_backups") {
		t.Fatalf("want both errors, got %v", err)
	}
}
