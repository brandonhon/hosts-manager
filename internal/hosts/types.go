package hosts

import (
	"time"
)

type Entry struct {
	IP        string   `json:"ip" yaml:"ip"`
	Hostnames []string `json:"hostnames" yaml:"hostnames"`
	Comment   string   `json:"comment,omitempty" yaml:"comment,omitempty"`
	Category  string   `json:"category" yaml:"category"`
	Enabled   bool     `json:"enabled" yaml:"enabled"`
	LineNum   int      `json:"line_num,omitempty" yaml:"line_num,omitempty"`
}

type Category struct {
	Name        string  `json:"name" yaml:"name"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled     bool    `json:"enabled" yaml:"enabled"`
	Entries     []Entry `json:"entries" yaml:"entries"`
}

type HostsFile struct {
	Categories []Category `json:"categories" yaml:"categories"`
	Header     []string   `json:"header,omitempty" yaml:"header,omitempty"`
	Footer     []string   `json:"footer,omitempty" yaml:"footer,omitempty"`
	Modified   time.Time  `json:"modified" yaml:"modified"`
	FilePath   string     `json:"file_path" yaml:"file_path"`
}

type Profile struct {
	Name        string     `json:"name" yaml:"name"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
	Categories  []Category `json:"categories" yaml:"categories"`
	Active      bool       `json:"active" yaml:"active"`
}

type BackupInfo struct {
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
	FilePath  string    `json:"file_path" yaml:"file_path"`
	Hash      string    `json:"hash" yaml:"hash"`
	Size      int64     `json:"size" yaml:"size"`
}

type Operation struct {
	Type        string      `json:"type" yaml:"type"`
	Target      string      `json:"target" yaml:"target"`
	OldValue    interface{} `json:"old_value,omitempty" yaml:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty" yaml:"new_value,omitempty"`
	Timestamp   time.Time   `json:"timestamp" yaml:"timestamp"`
	Description string      `json:"description" yaml:"description"`
}

type HistoryEntry struct {
	Operations []Operation `json:"operations" yaml:"operations"`
	Timestamp  time.Time   `json:"timestamp" yaml:"timestamp"`
	Hash       string      `json:"hash" yaml:"hash"`
	Message    string      `json:"message" yaml:"message"`
}

const (
	OpTypeAdd     = "add"
	OpTypeDelete  = "delete"
	OpTypeEnable  = "enable"
	OpTypeDisable = "disable"
	OpTypeUpdate  = "update"
	OpTypeComment = "comment"
)

const (
	CategoryDevelopment = "development"
	CategoryStaging     = "staging"
	CategoryProduction  = "production"
	CategoryCustom      = "custom"
	CategoryDefault     = "default"
)