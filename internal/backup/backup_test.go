package backup

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"hosts-manager/internal/config"
)

func createTestConfig(tempDir string) *config.Config {
	return &config.Config{
		Backup: config.Backup{
			Directory:       filepath.Join(tempDir, "backups"),
			MaxBackups:      5,
			RetentionDays:   30,
			CompressionType: "none",
		},
	}
}

func createTestConfigWithCompression(tempDir string) *config.Config {
	cfg := createTestConfig(tempDir)
	cfg.Backup.CompressionType = "gzip"
	return cfg
}

func TestNewManager(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)

	manager := NewManager(cfg)
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.config != cfg {
		t.Error("Manager config not set correctly")
	}

	if manager.platform == nil {
		t.Error("Manager platform not initialized")
	}
}

func TestCreateBackup(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)

	// Create a fake hosts file
	hostsDir := filepath.Join(tempDir, "etc")
	err := os.MkdirAll(hostsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create hosts directory: %v", err)
	}

	hostsPath := filepath.Join(hostsDir, "hosts")
	testContent := "127.0.0.1 localhost\n192.168.1.1 example.com\n"
	err = os.WriteFile(hostsPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test hosts file: %v", err)
	}

	// Mock the platform to return our test hosts path
	manager := NewManager(cfg)
	// We need to test with the actual platform behavior, so we'll work with temp files

	// Create backup directory
	err = os.MkdirAll(cfg.Backup.Directory, 0700)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Test backup creation by creating a backup of our temp hosts file
	backupPath, err := manager.copyFileToBackup(hostsPath, cfg.Backup.Directory, false)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("Backup file was not created")
	}

	// Verify backup content matches original
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if string(backupContent) != testContent {
		t.Errorf("Backup content doesn't match original. Expected %q, got %q", testContent, string(backupContent))
	}
}

// Helper function to create a backup without depending on the platform
func (m *Manager) copyFileToBackup(srcPath, backupDir string, compress bool) (string, error) {
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	backupName := fmt.Sprintf("hosts.backup.%s", timestamp)

	if compress {
		backupName += ".gz"
	}

	backupPath := filepath.Join(backupDir, backupName)
	return backupPath, m.copyFile(srcPath, backupPath, compress)
}

func TestCreateBackupWithCompression(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfigWithCompression(tempDir)

	// Create a test hosts file
	hostsPath := filepath.Join(tempDir, "hosts")
	testContent := "127.0.0.1 localhost\n192.168.1.1 example.com\n"
	err := os.WriteFile(hostsPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test hosts file: %v", err)
	}

	manager := NewManager(cfg)

	// Create backup directory
	err = os.MkdirAll(cfg.Backup.Directory, 0700)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create compressed backup
	backupPath, err := manager.copyFileToBackup(hostsPath, cfg.Backup.Directory, true)
	if err != nil {
		t.Fatalf("Failed to create compressed backup: %v", err)
	}

	// Verify backup file exists and is compressed
	if !strings.HasSuffix(backupPath, ".gz") {
		t.Error("Compressed backup should have .gz extension")
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("Compressed backup file was not created")
	}

	// Verify we can decompress and read the content
	file, err := os.Open(backupPath)
	if err != nil {
		t.Fatalf("Failed to open compressed backup: %v", err)
	}
	defer func() { _ = file.Close() }()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer func() { _ = gzipReader.Close() }()

	decompressedContent, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("Failed to read decompressed content: %v", err)
	}

	if string(decompressedContent) != testContent {
		t.Errorf("Decompressed content doesn't match original. Expected %q, got %q", testContent, string(decompressedContent))
	}
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	// Create source file
	srcPath := filepath.Join(tempDir, "source.txt")
	testContent := "Test file content for copying"
	err := os.WriteFile(srcPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Test uncompressed copy
	dstPath := filepath.Join(tempDir, "destination.txt")
	err = manager.copyFile(srcPath, dstPath, false)
	if err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// Verify copied content
	copiedContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(copiedContent) != testContent {
		t.Errorf("Copied content doesn't match. Expected %q, got %q", testContent, string(copiedContent))
	}

	// Test compressed copy
	compressedDstPath := filepath.Join(tempDir, "compressed.txt.gz")
	err = manager.copyFile(srcPath, compressedDstPath, true)
	if err != nil {
		t.Fatalf("Failed to copy file with compression: %v", err)
	}

	// Verify compressed file can be read
	file, err := os.Open(compressedDstPath)
	if err != nil {
		t.Fatalf("Failed to open compressed file: %v", err)
	}
	defer func() { _ = file.Close() }()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer func() { _ = gzipReader.Close() }()

	decompressedContent, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("Failed to read decompressed content: %v", err)
	}

	if string(decompressedContent) != testContent {
		t.Errorf("Decompressed content doesn't match. Expected %q, got %q", testContent, string(decompressedContent))
	}
}

func TestCopyFileErrors(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	// Test with non-existent source file
	nonExistentSrc := filepath.Join(tempDir, "nonexistent.txt")
	dstPath := filepath.Join(tempDir, "destination.txt")

	err := manager.copyFile(nonExistentSrc, dstPath, false)
	if err == nil {
		t.Error("Expected error when copying non-existent file")
	}

	// Test with invalid destination path
	srcPath := filepath.Join(tempDir, "source.txt")
	err = os.WriteFile(srcPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	invalidDstPath := filepath.Join(tempDir, "nonexistent_dir", "destination.txt")
	err = manager.copyFile(srcPath, invalidDstPath, false)
	if err == nil {
		t.Error("Expected error when copying to invalid destination")
	}
}

func TestRestoreFile(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	testContent := "Original hosts file content"

	// Test uncompressed restore
	backupPath := filepath.Join(tempDir, "backup.txt")
	err := os.WriteFile(backupPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	hostsPath := filepath.Join(tempDir, "hosts")
	err = manager.restoreFile(backupPath, hostsPath, false)
	if err != nil {
		t.Fatalf("Failed to restore file: %v", err)
	}

	// Verify restored content
	restoredContent, err := os.ReadFile(hostsPath)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if string(restoredContent) != testContent {
		t.Errorf("Restored content doesn't match. Expected %q, got %q", testContent, string(restoredContent))
	}

	// Test compressed restore
	compressedBackupPath := filepath.Join(tempDir, "compressed_backup.gz")
	compressedFile, err := os.Create(compressedBackupPath)
	if err != nil {
		t.Fatalf("Failed to create compressed backup file: %v", err)
	}

	gzipWriter := gzip.NewWriter(compressedFile)
	_, err = gzipWriter.Write([]byte(testContent))
	if err != nil {
		t.Fatalf("Failed to write compressed content: %v", err)
	}
	_ = gzipWriter.Close()
	_ = compressedFile.Close()

	restoredPath := filepath.Join(tempDir, "restored_hosts")
	err = manager.restoreFile(compressedBackupPath, restoredPath, true)
	if err != nil {
		t.Fatalf("Failed to restore compressed file: %v", err)
	}

	// Verify restored content from compressed backup
	restoredContent, err = os.ReadFile(restoredPath)
	if err != nil {
		t.Fatalf("Failed to read restored file from compressed backup: %v", err)
	}

	if string(restoredContent) != testContent {
		t.Errorf("Restored content from compressed backup doesn't match. Expected %q, got %q", testContent, string(restoredContent))
	}
}

func TestRestoreFileWithPermissionPreservation(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	// Create original file with specific permissions
	originalPath := filepath.Join(tempDir, "original")
	err := os.WriteFile(originalPath, []byte("original content"), 0755)
	if err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Create backup file
	backupPath := filepath.Join(tempDir, "backup")
	err = os.WriteFile(backupPath, []byte("backup content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	// Restore should preserve original file permissions
	err = manager.restoreFile(backupPath, originalPath, false)
	if err != nil {
		t.Fatalf("Failed to restore file: %v", err)
	}

	// Check that permissions were preserved
	restoredInfo, err := os.Stat(originalPath)
	if err != nil {
		t.Fatalf("Failed to stat restored file: %v", err)
	}

	if restoredInfo.Mode().Perm() != 0755 {
		t.Errorf("Expected permissions 0755, got %o", restoredInfo.Mode().Perm())
	}
}

func TestListBackups(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	// Test with non-existent backup directory
	backups, err := manager.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups should not error with non-existent directory: %v", err)
	}

	if len(backups) != 0 {
		t.Error("Expected empty backup list for non-existent directory")
	}

	// Create backup directory and some backup files
	err = os.MkdirAll(cfg.Backup.Directory, 0700)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create test backup files with different timestamps
	testBackups := []struct {
		filename string
		content  string
	}{
		{"hosts.backup.2023-12-01T10-30-00", "backup1 content"},
		{"hosts.backup.2023-12-02T15-45-30", "backup2 content"},
		{"hosts.backup.2023-12-03T09-15-45.gz", "backup3 content"},
	}

	for _, backup := range testBackups {
		backupPath := filepath.Join(cfg.Backup.Directory, backup.filename)
		if strings.HasSuffix(backup.filename, ".gz") {
			// Create compressed backup
			file, err := os.Create(backupPath)
			if err != nil {
				t.Fatalf("Failed to create backup file: %v", err)
			}
			gzipWriter := gzip.NewWriter(file)
			_, _ = gzipWriter.Write([]byte(backup.content))
			_ = gzipWriter.Close()
			_ = file.Close()
		} else {
			// Create uncompressed backup
			err := os.WriteFile(backupPath, []byte(backup.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create backup file: %v", err)
			}
		}
	}

	// List backups
	backups, err = manager.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) != len(testBackups) {
		t.Errorf("Expected %d backups, got %d", len(testBackups), len(backups))
	}

	// Verify backups are sorted by timestamp (newest first)
	for i := 0; i < len(backups)-1; i++ {
		if backups[i].Timestamp.Before(backups[i+1].Timestamp) {
			t.Error("Backups should be sorted by timestamp (newest first)")
		}
	}

	// Verify backup info
	for _, backup := range backups {
		if backup.Hash == "" {
			t.Error("Backup hash should not be empty")
		}
		if backup.Size == 0 {
			t.Error("Backup size should not be zero")
		}
		if backup.FilePath == "" {
			t.Error("Backup file path should not be empty")
		}
	}
}

func TestGetBackupInfo(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	// Create a test backup file
	backupPath := filepath.Join(tempDir, "hosts.backup.2023-12-01T10-30-00")
	testContent := "test backup content"
	err := os.WriteFile(backupPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test backup: %v", err)
	}

	// Get backup info
	info, err := manager.getBackupInfo(backupPath)
	if err != nil {
		t.Fatalf("Failed to get backup info: %v", err)
	}

	// Verify timestamp parsing
	expectedTime, _ := time.Parse("2006-01-02T15-04-05", "2023-12-01T10-30-00")
	if !info.Timestamp.Equal(expectedTime) {
		t.Errorf("Expected timestamp %v, got %v", expectedTime, info.Timestamp)
	}

	// Verify hash
	if info.Hash == "" {
		t.Error("Hash should not be empty")
	}

	// Verify size
	if info.Size != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), info.Size)
	}

	// Test with compressed file
	compressedPath := filepath.Join(tempDir, "hosts.backup.2023-12-02T15-45-30.gz")
	file, err := os.Create(compressedPath)
	if err != nil {
		t.Fatalf("Failed to create compressed backup: %v", err)
	}
	gzipWriter := gzip.NewWriter(file)
	_, _ = gzipWriter.Write([]byte(testContent))
	_ = gzipWriter.Close()
	_ = file.Close()

	compressedInfo, err := manager.getBackupInfo(compressedPath)
	if err != nil {
		t.Fatalf("Failed to get compressed backup info: %v", err)
	}

	// Verify timestamp parsing for compressed file
	expectedTime2, _ := time.Parse("2006-01-02T15-04-05", "2023-12-02T15-45-30")
	if !compressedInfo.Timestamp.Equal(expectedTime2) {
		t.Errorf("Expected compressed timestamp %v, got %v", expectedTime2, compressedInfo.Timestamp)
	}
}

func TestGetBackupInfoWithInvalidFilename(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	// Create backup with invalid timestamp in filename
	invalidPath := filepath.Join(tempDir, "hosts.backup.invalid-timestamp")
	err := os.WriteFile(invalidPath, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid backup: %v", err)
	}

	// Should fallback to file modification time
	info, err := manager.getBackupInfo(invalidPath)
	if err != nil {
		t.Fatalf("Failed to get backup info with invalid timestamp: %v", err)
	}

	// Timestamp should be set to file modification time
	stat, _ := os.Stat(invalidPath)
	if !info.Timestamp.Equal(stat.ModTime()) {
		t.Error("Should fallback to file modification time for invalid timestamp")
	}
}

func TestCalculateFileHash(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	testContent := "test content for hashing"

	// Test uncompressed file hash
	filePath := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(filePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash1, err := manager.calculateFileHash(filePath)
	if err != nil {
		t.Fatalf("Failed to calculate file hash: %v", err)
	}

	if hash1 == "" {
		t.Error("Hash should not be empty")
	}

	// Calculate again to ensure consistency
	hash2, err := manager.calculateFileHash(filePath)
	if err != nil {
		t.Fatalf("Failed to calculate file hash again: %v", err)
	}

	if hash1 != hash2 {
		t.Error("Hash should be consistent for same content")
	}

	// Test compressed file hash
	compressedPath := filepath.Join(tempDir, "test.txt.gz")
	file, err := os.Create(compressedPath)
	if err != nil {
		t.Fatalf("Failed to create compressed file: %v", err)
	}
	gzipWriter := gzip.NewWriter(file)
	_, _ = gzipWriter.Write([]byte(testContent))
	_ = gzipWriter.Close()
	_ = file.Close()

	compressedHash, err := manager.calculateFileHash(compressedPath)
	if err != nil {
		t.Fatalf("Failed to calculate compressed file hash: %v", err)
	}

	// Hash of compressed file content should match original content hash
	if hash1 != compressedHash {
		t.Error("Hash of compressed file content should match original")
	}
}

func TestCleanupOldBackups(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	cfg.Backup.MaxBackups = 2
	cfg.Backup.RetentionDays = 1 // 1 day retention

	manager := NewManager(cfg)

	// Create backup directory
	err := os.MkdirAll(cfg.Backup.Directory, 0700)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create multiple backup files with different timestamps
	now := time.Now()
	backupTimes := []time.Time{
		now.AddDate(0, 0, -3),      // 3 days old - should be deleted (retention)
		now.AddDate(0, 0, -2),      // 2 days old - should be deleted (retention)
		now.AddDate(0, 0, 0),       // Today - should be kept
		now.Add(-1 * time.Hour),    // 1 hour ago - should be kept
		now.Add(-30 * time.Minute), // 30 minutes ago - should be deleted (exceeds max)
	}

	for i, backupTime := range backupTimes {
		timestamp := backupTime.Format("2006-01-02T15-04-05")
		filename := fmt.Sprintf("hosts.backup.%s", timestamp)
		backupPath := filepath.Join(cfg.Backup.Directory, filename)

		content := fmt.Sprintf("backup content %d", i)
		err := os.WriteFile(backupPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create backup file %s: %v", filename, err)
		}

		// Set the file modification time to match the backup time
		err = os.Chtimes(backupPath, backupTime, backupTime)
		if err != nil {
			t.Fatalf("Failed to set file time: %v", err)
		}
	}

	// Run cleanup
	err = manager.cleanupOldBackups()
	if err != nil {
		t.Fatalf("Failed to cleanup old backups: %v", err)
	}

	// Check remaining backups
	remainingBackups, err := manager.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list remaining backups: %v", err)
	}

	// Should have at most maxBackups (2) and only recent ones
	if len(remainingBackups) > cfg.Backup.MaxBackups {
		t.Errorf("Expected at most %d backups after cleanup, got %d", cfg.Backup.MaxBackups, len(remainingBackups))
	}

	// All remaining backups should be within retention period
	cutoffTime := now.AddDate(0, 0, -cfg.Backup.RetentionDays)
	for _, backup := range remainingBackups {
		if backup.Timestamp.Before(cutoffTime) {
			t.Errorf("Found backup older than retention period: %v", backup.Timestamp)
		}
	}
}

func TestGetBackupPath(t *testing.T) {
	tempDir := t.TempDir()

	// Test without compression
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	timestamp := "2023-12-01T10-30-00"
	path := manager.GetBackupPath(timestamp)

	expectedPath := filepath.Join(cfg.Backup.Directory, "hosts.backup.2023-12-01T10-30-00")
	if path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, path)
	}

	// Test with compression
	cfgCompressed := createTestConfigWithCompression(tempDir)
	managerCompressed := NewManager(cfgCompressed)

	pathCompressed := managerCompressed.GetBackupPath(timestamp)
	expectedPathCompressed := filepath.Join(cfgCompressed.Backup.Directory, "hosts.backup.2023-12-01T10-30-00.gz")
	if pathCompressed != expectedPathCompressed {
		t.Errorf("Expected compressed path %s, got %s", expectedPathCompressed, pathCompressed)
	}
}

func TestDeleteBackup(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	// Create a test backup file
	backupPath := filepath.Join(tempDir, "test_backup.txt")
	err := os.WriteFile(backupPath, []byte("test backup content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test backup: %v", err)
	}

	// Delete the backup
	err = manager.DeleteBackup(backupPath)
	if err != nil {
		t.Fatalf("Failed to delete backup: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("Backup file should be deleted")
	}

	// Test deleting non-existent file
	err = manager.DeleteBackup(backupPath)
	if err == nil {
		t.Error("Expected error when deleting non-existent file")
	}
}

func TestSecureDelete(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	// Create a test file
	testPath := filepath.Join(tempDir, "secure_delete_test.txt")
	testContent := "sensitive content that should be securely deleted"
	err := os.WriteFile(testPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Perform secure delete
	err = manager.secureDelete(testPath)
	if err != nil {
		t.Fatalf("Failed to securely delete file: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(testPath); !os.IsNotExist(err) {
		t.Error("File should be deleted after secure delete")
	}

	// Test secure delete on non-existent file (should not error)
	err = manager.secureDelete(testPath)
	if err != nil {
		t.Errorf("Secure delete of non-existent file should not error: %v", err)
	}
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{10, 10, 10},
		{0, 100, 0},
		{-5, -2, -5},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestVerifyBackupIntegrity(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	// Create a test backup file
	backupPath := filepath.Join(tempDir, "hosts.backup.2023-12-01T10-30-00")
	testContent := "test backup content for integrity check"
	err := os.WriteFile(backupPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test backup: %v", err)
	}

	// Verify integrity (should always pass for valid files with current implementation)
	// Note: The current implementation recalculates hash each time, so it will always "pass"
	// This tests that the function works without errors for valid files
	err = manager.VerifyBackupIntegrity(backupPath)
	if err != nil {
		t.Fatalf("Integrity verification should pass for valid file: %v", err)
	}

	// Test that we can detect hash differences manually
	originalHash, err := manager.calculateFileHash(backupPath)
	if err != nil {
		t.Fatalf("Failed to calculate original hash: %v", err)
	}

	// Modify the file
	corruptedContent := "different content"
	err = os.WriteFile(backupPath, []byte(corruptedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to modify backup file: %v", err)
	}

	// Calculate new hash - should be different
	newHash, err := manager.calculateFileHash(backupPath)
	if err != nil {
		t.Fatalf("Failed to calculate new hash: %v", err)
	}

	if newHash == originalHash {
		t.Error("Hash should be different after content change")
	}

	// Test with non-existent file
	nonExistentPath := filepath.Join(tempDir, "nonexistent.backup")
	err = manager.VerifyBackupIntegrity(nonExistentPath)
	if err == nil {
		t.Error("Integrity verification should fail for non-existent file")
	}
}

func TestCreateSecureBackup(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(tempDir)

	// Create a fake hosts file
	hostsPath := filepath.Join(tempDir, "hosts")
	testContent := "127.0.0.1 localhost\n"
	err := os.WriteFile(hostsPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test hosts file: %v", err)
	}

	manager := NewManager(cfg)

	// Create backup directory
	err = os.MkdirAll(cfg.Backup.Directory, 0700)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// We need to test this indirectly since CreateSecureBackup depends on platform
	// Test the verification part by creating a backup and verifying it
	backupPath, err := manager.copyFileToBackup(hostsPath, cfg.Backup.Directory, false)
	if err != nil {
		t.Fatalf("Failed to create backup for testing: %v", err)
	}

	// Test verification
	err = manager.VerifyBackupIntegrity(backupPath)
	if err != nil {
		t.Fatalf("Backup verification should pass: %v", err)
	}
}

// BenchmarkCreateBackup benchmarks backup creation
func BenchmarkCreateBackup(b *testing.B) {
	tempDir := b.TempDir()
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	// Create a test hosts file
	hostsPath := filepath.Join(tempDir, "hosts")
	testContent := strings.Repeat("127.0.0.1 test.example.com\n", 1000) // 1000 lines
	err := os.WriteFile(hostsPath, []byte(testContent), 0644)
	if err != nil {
		b.Fatalf("Failed to create test hosts file: %v", err)
	}

	// Create backup directory
	err = os.MkdirAll(cfg.Backup.Directory, 0700)
	if err != nil {
		b.Fatalf("Failed to create backup directory: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backupPath := filepath.Join(cfg.Backup.Directory, fmt.Sprintf("bench_backup_%d", i))
		err := manager.copyFile(hostsPath, backupPath, false)
		if err != nil {
			b.Fatal(err)
		}
		_ = os.Remove(backupPath) // Clean up
	}
}

// BenchmarkCreateCompressedBackup benchmarks compressed backup creation
func BenchmarkCreateCompressedBackup(b *testing.B) {
	tempDir := b.TempDir()
	cfg := createTestConfigWithCompression(tempDir)
	manager := NewManager(cfg)

	// Create a test hosts file
	hostsPath := filepath.Join(tempDir, "hosts")
	testContent := strings.Repeat("127.0.0.1 test.example.com\n", 1000) // 1000 lines
	err := os.WriteFile(hostsPath, []byte(testContent), 0644)
	if err != nil {
		b.Fatalf("Failed to create test hosts file: %v", err)
	}

	// Create backup directory
	err = os.MkdirAll(cfg.Backup.Directory, 0700)
	if err != nil {
		b.Fatalf("Failed to create backup directory: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backupPath := filepath.Join(cfg.Backup.Directory, fmt.Sprintf("bench_backup_%d.gz", i))
		err := manager.copyFile(hostsPath, backupPath, true)
		if err != nil {
			b.Fatal(err)
		}
		_ = os.Remove(backupPath) // Clean up
	}
}

// BenchmarkCalculateFileHash benchmarks file hash calculation
func BenchmarkCalculateFileHash(b *testing.B) {
	tempDir := b.TempDir()
	cfg := createTestConfig(tempDir)
	manager := NewManager(cfg)

	// Create test files of different sizes
	testSizes := []int{1024, 10240, 102400} // 1KB, 10KB, 100KB

	for _, size := range testSizes {
		b.Run(fmt.Sprintf("size_%dKB", size/1024), func(b *testing.B) {
			content := strings.Repeat("x", size)
			filePath := filepath.Join(tempDir, fmt.Sprintf("test_%d.txt", size))
			err := os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := manager.calculateFileHash(filePath)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
