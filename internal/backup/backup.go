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

	"github.com/brandonhon/hosts-manager/internal/config"
	"github.com/brandonhon/hosts-manager/internal/hosts"
	"github.com/brandonhon/hosts-manager/pkg/platform"
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
	// Callers pass either a glob result from the backup directory or a path
	// already validated against it. VerifyBackupIntegrity is exported, so a
	// future caller must preserve that invariant.
	// #nosec G304 -- backup-directory path, validated or enumerated
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	var reader io.Reader = file
	if strings.HasSuffix(filePath, ".gz") {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return "", err
		}
		defer func() { _ = gzipReader.Close() }()
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
		if err := m.secureDelete(filePath); err != nil {
			fmt.Printf("Warning: failed to securely remove old backup %s: %v\n", filePath, err)
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

	return m.secureDelete(filePath)
}

// secureDelete overwrites file content before deletion for security
func (m *Manager) secureDelete(filePath string) error {
	// Get file info first
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	fileSize := fileInfo.Size()

	// Open file for writing
	// Reached from cleanupOldBackups with a glob result, or DeleteBackup with
	// a caller-supplied backup path. Exported entry points must keep that
	// invariant, since this overwrites the file before removing it.
	// #nosec G304 -- backup-directory path, validated or enumerated
	file, err := os.OpenFile(filePath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open file for secure deletion: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Overwrite with zeros (single pass is sufficient for most cases)
	zeroBuffer := make([]byte, min(4096, int(fileSize))) // 4KB chunks
	for i := int64(0); i < fileSize; i += int64(len(zeroBuffer)) {
		remaining := fileSize - i
		if remaining < int64(len(zeroBuffer)) {
			zeroBuffer = zeroBuffer[:remaining]
		}

		if _, err := file.WriteAt(zeroBuffer, i); err != nil {
			return fmt.Errorf("failed to overwrite file content: %w", err)
		}
	}

	// Sync to ensure data is written to disk
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync overwritten data: %w", err)
	}

	// Close before removing
	_ = file.Close()

	// Now remove the file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove file after overwriting: %w", err)
	}

	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// VerifyBackupIntegrity checks that a backup can still be read back in full:
// it must be non-empty, and a compressed backup must decompress cleanly, which
// is where a truncated or corrupt archive fails.
//
// It cannot detect deliberate tampering. No hash is recorded when a backup is
// taken, so there is nothing to compare a later hash against — an attacker who
// replaced a backup wholesale would produce a file that reads back perfectly.
// Detecting that would need the hash stored out of band at creation time.
//
// This previously hashed the file, then hashed the same file again through
// getBackupInfo and compared the two — a comparison that was true by
// construction and could never fail. It did reject corrupt archives, but only
// incidentally, because both reads had to succeed before the pointless
// comparison happened. An empty backup passed. The check is now the one that
// was being performed by accident, stated deliberately.
func (m *Manager) VerifyBackupIntegrity(filePath string) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat backup: %w", err)
	}
	if stat.Size() == 0 {
		return fmt.Errorf("backup is empty: %s", filePath)
	}

	// calculateFileHash reads the file end to end, decompressing gzip backups
	// as it goes, so a truncated or corrupt archive surfaces here.
	if _, err := m.calculateFileHash(filePath); err != nil {
		return fmt.Errorf("backup is unreadable: %w", err)
	}

	return nil
}

// CreateSecureBackup creates a backup and verifies that its contents match the
// hosts file it was taken from, deleting it if they do not.
//
// The comparison is against the source rather than against the backup itself,
// which is the only reference available that makes the check mean anything. It
// catches a partial or truncated copy — a real risk, since copyFile discards
// the error from closing the gzip writer, so a failed flush would otherwise
// leave a short backup that looks fine.
//
// The hosts file is read again after the copy, so a change to it in that window
// shows up as a mismatch. That is the intended outcome: a backup taken while
// the file was being modified should not be trusted.
func (m *Manager) CreateSecureBackup() (string, error) {
	backupPath, err := m.CreateBackup()
	if err != nil {
		return "", err
	}

	sourceHash, err := m.calculateFileHash(m.platform.GetHostsFilePath())
	if err != nil {
		_ = m.secureDelete(backupPath)
		return "", fmt.Errorf("backup verification failed: cannot hash hosts file: %w", err)
	}

	backupHash, err := m.calculateFileHash(backupPath)
	if err != nil {
		_ = m.secureDelete(backupPath)
		return "", fmt.Errorf("backup verification failed: %w", err)
	}

	if backupHash != sourceHash {
		_ = m.secureDelete(backupPath)
		return "", fmt.Errorf("backup verification failed: contents do not match %s",
			m.platform.GetHostsFilePath())
	}

	return backupPath, nil
}
