package backup

import (
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"hosts-manager/internal/config"
	"hosts-manager/pkg/platform"
)

type Manager struct {
	config   *config.Config
	platform *platform.Platform
}

type BackupInfo struct {
	Timestamp time.Time `json:"timestamp"`
	FilePath  string    `json:"file_path"`
	Hash      string    `json:"hash"`
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
	if err := os.MkdirAll(backupDir, 0755); err != nil {
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

	m.cleanupOldBackups()

	return backupPath, nil
}

func (m *Manager) copyFile(src, dst string, compress bool) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if compress {
		gzipWriter := gzip.NewWriter(dstFile)
		defer gzipWriter.Close()
		_, err = io.Copy(gzipWriter, srcFile)
	} else {
		_, err = io.Copy(dstFile, srcFile)
	}

	return err
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

func (m *Manager) restoreFile(src, dst string, decompress bool) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	var reader io.Reader = srcFile
	if decompress {
		gzipReader, err := gzip.NewReader(srcFile)
		if err != nil {
			return err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	_, err = io.Copy(dstFile, reader)
	return err
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

	hash, err := m.calculateFileHash(filePath)
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
		Hash:      hash,
		Size:      stat.Size(),
	}, nil
}

func (m *Manager) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var reader io.Reader = file
	if strings.HasSuffix(filePath, ".gz") {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return "", err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, reader); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
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

func (m *Manager) GetBackupPath(timestamp string) string {
	backupName := fmt.Sprintf("hosts.backup.%s", timestamp)
	if m.config.Backup.CompressionType == "gzip" {
		backupName += ".gz"
	}
	return filepath.Join(m.config.Backup.Directory, backupName)
}

func (m *Manager) DeleteBackup(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", filePath)
	}

	return os.Remove(filePath)
}