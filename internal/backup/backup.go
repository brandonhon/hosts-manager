package backup

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/brandonhon/hosts-manager/internal/config"
	"github.com/brandonhon/hosts-manager/internal/hosts"
	"github.com/brandonhon/hosts-manager/internal/platform"
)

type Manager struct {
	config   *config.Config
	platform *platform.Platform
}

type BackupInfo struct {
	Timestamp time.Time `json:"timestamp"`
	FilePath  string    `json:"file_path"`
	Size      int64     `json:"size"`
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config:   cfg,
		platform: platform.New(),
	}
}

func (m *Manager) CreateBackup() (string, error) {
	hostsPath := m.platform.GetHostsFilePath()

	if _, err := os.Stat(hostsPath); os.IsNotExist(err) {
		return "", fmt.Errorf("hosts file does not exist: %s", hostsPath)
	}

	backupDir := m.config.Backup.Directory
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02T15-04-05")
	backupName := fmt.Sprintf("hosts.backup.%s", timestamp)

	if m.config.Backup.CompressionType == "gzip" {
		backupName += ".gz"
	}

	backupPath := filepath.Join(backupDir, backupName)

	if err := m.copyFile(hostsPath, backupPath, m.config.Backup.CompressionType == "gzip"); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	_ = m.cleanupOldBackups()

	return backupPath, nil
}

func (m *Manager) copyFile(src, dst string, compress bool) error {
	// src is the platform hosts path and dst the backup path this package
	// composed from the configured backup directory. Neither is user input.
	// #nosec G304 -- internally composed paths
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	// #nosec G304 -- backup path composed from the configured backup directory
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	// Closing is where buffered data is flushed, so both closes are checked
	// rather than deferred and discarded. A backup that failed to flush is
	// short, and silently returning nil for it is how a truncated backup ends
	// up looking like a good one.
	if compress {
		gzipWriter := gzip.NewWriter(dstFile)
		if _, err := io.Copy(gzipWriter, srcFile); err != nil {
			_ = gzipWriter.Close()
			_ = dstFile.Close()
			return err
		}
		if err := gzipWriter.Close(); err != nil {
			_ = dstFile.Close()
			return fmt.Errorf("failed to finish compressing backup: %w", err)
		}
	} else {
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			_ = dstFile.Close()
			return err
		}
	}

	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("failed to close backup: %w", err)
	}
	return nil
}

func (m *Manager) RestoreBackup(backupPath string) error {
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	hostsPath := m.platform.GetHostsFilePath()

	currentBackupPath, err := m.CreateBackup()
	if err != nil {
		return fmt.Errorf("failed to create current backup before restore: %w", err)
	}

	isCompressed := strings.HasSuffix(backupPath, ".gz")

	if err := m.restoreFile(backupPath, hostsPath, isCompressed); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	fmt.Printf("Backup restored successfully. Previous version backed up to: %s\n", currentBackupPath)
	return nil
}

// restoreFile writes a backup back over the hosts file.
//
// This goes through hosts.AtomicWrite — the same lock, temp file and rename
// that every other write to the hosts file uses. It previously opened the
// destination with O_TRUNC and copied into it in place, so an interrupted or
// failed restore left a partially written hosts file where a valid one had
// been. Restore is the one operation you reach for when the hosts file is
// already wrong, which is the worst possible moment to be able to corrupt it.
//
// File permissions are preserved by AtomicWrite, which chmods the temp file to
// match the target before renaming.
func (m *Manager) restoreFile(src, dst string, decompress bool) error {
	// src reaches here from restoreCmd via validateFilePath, which requires
	// it to resolve inside the configured backup directory.
	// #nosec G304 -- path validated against the backup directory
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	var reader io.Reader = srcFile
	if decompress {
		gzipReader, err := gzip.NewReader(srcFile)
		if err != nil {
			return err
		}
		defer func() { _ = gzipReader.Close() }()
		reader = gzipReader
	}

	return hosts.AtomicWrite(dst, func(w io.Writer) error {
		_, err := io.Copy(w, reader)
		return err
	})
}

func (m *Manager) ListBackups() ([]BackupInfo, error) {
	backupDir := m.config.Backup.Directory

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}

	files, err := filepath.Glob(filepath.Join(backupDir, "hosts.backup.*"))
	if err != nil {
		return nil, fmt.Errorf("failed to list backup files: %w", err)
	}

	var backups []BackupInfo
	for _, file := range files {
		info, err := m.getBackupInfo(file)
		if err != nil {
			continue
		}
		backups = append(backups, info)
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

func (m *Manager) getBackupInfo(filePath string) (BackupInfo, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return BackupInfo{}, err
	}

	filename := filepath.Base(filePath)
	var timestampStr string

	if strings.HasSuffix(filename, ".gz") {
		timestampStr = strings.TrimSuffix(strings.TrimPrefix(filename, "hosts.backup."), ".gz")
	} else {
		timestampStr = strings.TrimPrefix(filename, "hosts.backup.")
	}

	timestamp, err := time.Parse("2006-01-02T15-04-05", timestampStr)
	if err != nil {
		timestamp = stat.ModTime()
	}

	return BackupInfo{
		Timestamp: timestamp,
		FilePath:  filePath,
		Size:      stat.Size(),
	}, nil
}

func (m *Manager) cleanupOldBackups() error {
	backups, err := m.ListBackups()
	if err != nil {
		return err
	}

	maxBackups := m.config.Backup.MaxBackups
	retentionDays := m.config.Backup.RetentionDays
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	var toDelete []string

	if len(backups) > maxBackups {
		for i := maxBackups; i < len(backups); i++ {
			toDelete = append(toDelete, backups[i].FilePath)
		}
	}

	for _, backup := range backups {
		if backup.Timestamp.Before(cutoffTime) {
			found := false
			for _, path := range toDelete {
				if path == backup.FilePath {
					found = true
					break
				}
			}
			if !found {
				toDelete = append(toDelete, backup.FilePath)
			}
		}
	}

	for _, filePath := range toDelete {
		if err := os.Remove(filePath); err != nil {
			fmt.Printf("Warning: failed to remove old backup %s: %v\n", filePath, err)
		}
	}

	return nil
}
