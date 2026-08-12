package audit

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotateIfNeeded(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 100, // Small size to trigger rotation
		maxLogs:    3,
	}

	// Test with non-existent file
	err := logger.rotateIfNeeded()
	if err != nil {
		t.Fatalf("rotateIfNeeded should not fail with non-existent file: %v", err)
	}

	// Create a small log file that doesn't need rotation
	smallContent := "small log content"
	err = os.WriteFile(logPath, []byte(smallContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write small log file: %v", err)
	}

	err = logger.rotateIfNeeded()
	if err != nil {
		t.Fatalf("rotateIfNeeded should not fail with small file: %v", err)
	}

	// Verify file still exists and wasn't rotated
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Small log file should not be rotated")
	}

	// Create a large log file that needs rotation
	largeContent := strings.Repeat("x", 200) // Larger than maxLogSize
	err = os.WriteFile(logPath, []byte(largeContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write large log file: %v", err)
	}

	err = logger.rotateIfNeeded()
	if err != nil {
		t.Fatalf("rotateIfNeeded failed with large file: %v", err)
	}

	// Verify original file was rotated
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("Large log file should be rotated away")
	}

	// Verify rotated file exists (compressed)
	rotatedPath := filepath.Join(tempDir, "audit.log.1.gz")
	if _, err := os.Stat(rotatedPath); os.IsNotExist(err) {
		t.Error("Rotated log file should exist")
	}
}

func TestRotateLog(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 100,
		maxLogs:    3,
	}

	// Create original log file
	originalContent := "original log content"
	err := os.WriteFile(logPath, []byte(originalContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write original log file: %v", err)
	}

	// Perform rotation
	err = logger.rotateLog()
	if err != nil {
		t.Fatalf("rotateLog failed: %v", err)
	}

	// Verify original file is gone
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("Original log file should be moved")
	}

	// Verify rotated file exists (compressed)
	rotatedPath := filepath.Join(tempDir, "audit.log.1.gz")
	if _, err := os.Stat(rotatedPath); os.IsNotExist(err) {
		t.Error("Rotated compressed log file should exist")
	}

	// Verify compressed content matches original
	compressedFile, err := os.Open(rotatedPath)
	if err != nil {
		t.Fatalf("Failed to open compressed file: %v", err)
	}
	defer func() { _ = compressedFile.Close() }()

	gzipReader, err := gzip.NewReader(compressedFile)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer func() { _ = gzipReader.Close() }()

	decompressedContent, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("Failed to read decompressed content: %v", err)
	}

	if string(decompressedContent) != originalContent {
		t.Errorf("Decompressed content %q doesn't match original %q", string(decompressedContent), originalContent)
	}
}

func TestRotateLogMultiple(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 100,
		maxLogs:    3,
	}

	// Create and rotate multiple logs
	for i := 1; i <= 5; i++ {
		content := fmt.Sprintf("log content %d", i)
		err := os.WriteFile(logPath, []byte(content), 0600)
		if err != nil {
			t.Fatalf("Failed to write log file %d: %v", i, err)
		}

		err = logger.rotateLog()
		if err != nil {
			t.Fatalf("rotateLog failed for iteration %d: %v", i, err)
		}
	}

	// Rotating more times than maxLogs must leave exactly maxLogs rotated
	// files. This previously asserted only "at least one, at most maxLogs",
	// which passed while rotation was in fact keeping exactly one: shifting
	// matched only the uncompressed name, so every slot beyond .1 was missed.
	rotatedCount := 0
	for i := 1; i <= logger.maxLogs; i++ {
		rotatedPath := filepath.Join(tempDir, fmt.Sprintf("audit.log.%d.gz", i))
		if _, err := os.Stat(rotatedPath); err == nil {
			rotatedCount++
		} else {
			t.Errorf("rotated log %s is missing", filepath.Base(rotatedPath))
		}
	}

	if rotatedCount != logger.maxLogs {
		t.Errorf("kept %d rotated files, want exactly %d", rotatedCount, logger.maxLogs)
	}

	// Verify older logs beyond maxLogs don't exist
	oldLogPath := filepath.Join(tempDir, fmt.Sprintf("audit.log.%d.gz", logger.maxLogs+1))
	if _, err := os.Stat(oldLogPath); !os.IsNotExist(err) {
		t.Error("Old log beyond maxLogs should be removed")
	}

	// Verify most recent rotated log has expected content
	recentPath := filepath.Join(tempDir, "audit.log.1.gz")
	file, err := os.Open(recentPath)
	if err != nil {
		t.Fatalf("Failed to open recent rotated log: %v", err)
	}
	defer func() { _ = file.Close() }()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer func() { _ = gzipReader.Close() }()

	content, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}

	expectedContent := "log content 5"
	if string(content) != expectedContent {
		t.Errorf("Most recent log content %q doesn't match expected %q", string(content), expectedContent)
	}
}

func TestCompressLog(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	logger := &Logger{
		logPath:    filepath.Join(tempDir, "audit.log"),
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 100,
		maxLogs:    3,
	}

	// Test with normal file
	testContent := "This is test log content that will be compressed"
	err := os.WriteFile(logPath, []byte(testContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write test log: %v", err)
	}

	err = logger.compressLog(logPath)
	if err != nil {
		t.Fatalf("compressLog failed: %v", err)
	}

	// Verify original file is removed
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("Original log file should be removed after compression")
	}

	// Verify compressed file exists
	compressedPath := logPath + ".gz"
	if _, err := os.Stat(compressedPath); os.IsNotExist(err) {
		t.Error("Compressed file should exist")
	}

	// Verify compressed content
	file, err := os.Open(compressedPath)
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
		t.Errorf("Decompressed content doesn't match original")
	}
}

func TestCompressLogNonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "nonexistent.log")

	logger := &Logger{
		logPath:    filepath.Join(tempDir, "audit.log"),
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 100,
		maxLogs:    3,
	}

	err := logger.compressLog(logPath)
	if err == nil {
		t.Error("compressLog should fail with non-existent file")
	}
}

func TestCompressLogLargeFile(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "large.log")

	logger := &Logger{
		logPath:    filepath.Join(tempDir, "audit.log"),
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 100,
		maxLogs:    3,
	}

	// Create a file larger than the compression limit
	// We'll simulate this by creating a file and then testing the size check
	largeContent := strings.Repeat("x", 1000) // 1KB file, small enough for test
	err := os.WriteFile(logPath, []byte(largeContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write large test file: %v", err)
	}

	// Should succeed with 1KB file
	err = logger.compressLog(logPath)
	if err != nil {
		t.Errorf("compressLog should succeed with small file: %v", err)
	}
}

func TestCompressLogStreamingAndChunking(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "chunked.log")

	logger := &Logger{
		logPath:    filepath.Join(tempDir, "audit.log"),
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 100,
		maxLogs:    3,
	}

	// Create content that's larger than the buffer size used in compression
	// The buffer size is 64KB, so create a 128KB file
	chunkSize := 1024
	totalChunks := 128
	var contentBuilder strings.Builder
	for i := 0; i < totalChunks; i++ {
		chunk := fmt.Sprintf("Chunk %03d: %s\n", i, strings.Repeat("x", chunkSize-20))
		contentBuilder.WriteString(chunk)
	}
	largeContent := contentBuilder.String()

	err := os.WriteFile(logPath, []byte(largeContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write large chunked file: %v", err)
	}

	err = logger.compressLog(logPath)
	if err != nil {
		t.Fatalf("compressLog failed with large file: %v", err)
	}

	// Verify compression worked correctly by decompressing and checking content
	compressedPath := logPath + ".gz"
	file, err := os.Open(compressedPath)
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

	if string(decompressedContent) != largeContent {
		t.Error("Decompressed large content doesn't match original")
	}

	// Verify compression actually reduced size
	originalInfo, _ := os.Stat(compressedPath)
	if originalInfo.Size() >= int64(len(largeContent)) {
		t.Error("Compressed file should be smaller than original")
	}
}

func TestLogRotationIntegration(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 200, // Small size to trigger rotation quickly
		maxLogs:    2,
	}

	// Log events until rotation occurs
	event := AuditEvent{
		EventType: EventHostsAdd,
		Severity:  SeverityInfo,
		Operation: "test_operation_with_long_name_to_increase_size",
		Resource:  "test_resource_with_long_name_to_increase_size",
		Success:   true,
		Details: map[string]interface{}{
			"key1": "value1_with_extra_content",
			"key2": "value2_with_extra_content",
			"key3": "value3_with_extra_content",
		},
	}

	// Log enough events to trigger multiple rotations
	for i := 0; i < 10; i++ {
		event.Operation = fmt.Sprintf("operation_%d_with_long_name_to_increase_size", i)
		err := logger.Log(event)
		if err != nil {
			t.Fatalf("Failed to log event %d: %v", i, err)
		}
	}

	// Verify that rotation occurred
	rotatedFiles := 0
	for i := 1; i <= logger.maxLogs; i++ {
		rotatedPath := filepath.Join(tempDir, fmt.Sprintf("audit.log.%d.gz", i))
		if _, err := os.Stat(rotatedPath); err == nil {
			rotatedFiles++
		}
	}

	if rotatedFiles == 0 {
		t.Error("Expected at least one rotated file")
	}

	// Verify current log file exists (it may or may not be smaller than maxLogSize
	// depending on timing of when the last event was logged vs when rotation occurred)
	if _, err := os.Stat(logPath); err != nil && !os.IsNotExist(err) {
		t.Errorf("Error checking current log file: %v", err)
	}

	// If current log exists, log one more event to verify logging still works
	if _, err := os.Stat(logPath); err == nil {
		testEvent := AuditEvent{
			EventType: EventHostsAdd,
			Severity:  SeverityInfo,
			Operation: "post_rotation_test",
			Resource:  "test_resource",
			Success:   true,
		}

		err = logger.Log(testEvent)
		if err != nil {
			t.Errorf("Failed to log after rotation: %v", err)
		}
	}
}

func TestLogRotationWithWriteFailure(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 50,
		maxLogs:    2,
	}

	// Create original log file
	originalContent := strings.Repeat("x", 100) // Larger than maxLogSize
	err := os.WriteFile(logPath, []byte(originalContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write original log file: %v", err)
	}

	// Make directory read-only to cause rotation failure
	if err := os.Chmod(tempDir, 0500); err != nil {
		t.Fatalf("Failed to change directory permissions: %v", err)
	}
	defer func() { _ = os.Chmod(tempDir, 0700) }() // Restore permissions

	// This should handle rotation failure gracefully
	err = logger.rotateIfNeeded()
	// The function should not return an error because rotation failures
	// are handled with warnings, not fatal errors
	if err != nil {
		t.Logf("Rotation failed as expected: %v", err)
	}

	// Restore permissions
	_ = os.Chmod(tempDir, 0700)
}

// BenchmarkRotateLog benchmarks log rotation
func BenchmarkRotateLog(b *testing.B) {
	tempDir := b.TempDir()

	for i := 0; i < b.N; i++ {
		// Create a unique logger for each iteration
		logPath := filepath.Join(tempDir, fmt.Sprintf("audit_%d.log", i))
		logger := &Logger{
			logPath:    logPath,
			enabled:    true,
			minLevel:   SeverityInfo,
			maxLogSize: 100,
			maxLogs:    3,
		}

		// Create log file to rotate
		content := strings.Repeat("log content ", 20)
		err := os.WriteFile(logPath, []byte(content), 0600)
		if err != nil {
			b.Fatal(err)
		}

		b.StartTimer()
		err = logger.rotateLog()
		b.StopTimer()

		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCompressLog benchmarks log compression
func BenchmarkCompressLog(b *testing.B) {
	tempDir := b.TempDir()

	// Create test content of various sizes
	testSizes := []int{1024, 10240, 102400} // 1KB, 10KB, 100KB

	for _, size := range testSizes {
		b.Run(fmt.Sprintf("size_%dKB", size/1024), func(b *testing.B) {
			content := strings.Repeat("x", size)

			for i := 0; i < b.N; i++ {
				logPath := filepath.Join(tempDir, fmt.Sprintf("bench_%d_%d.log", size, i))
				logger := &Logger{
					logPath:    filepath.Join(tempDir, "audit.log"),
					enabled:    true,
					minLevel:   SeverityInfo,
					maxLogSize: 100,
					maxLogs:    3,
				}

				// Write test file
				err := os.WriteFile(logPath, []byte(content), 0600)
				if err != nil {
					b.Fatal(err)
				}

				b.StartTimer()
				err = logger.compressLog(logPath)
				b.StopTimer()

				if err != nil {
					b.Fatal(err)
				}

				// Clean up compressed file
				_ = os.Remove(logPath + ".gz")
			}
		})
	}
}

// TestRotationRetainsMaxLogs is the regression test for the retention bug:
// rotated logs are compressed, but the shift step matched only the
// uncompressed name, so no slot was ever found, every shift was a no-op, and
// audit.log.1.gz was overwritten on each rotation. Whatever maxLogs said, one
// rotated log survived.
func TestRotationRetainsMaxLogs(t *testing.T) {
	const maxLogs = 5
	tempDir := t.TempDir()
	logger := &Logger{
		logPath:    filepath.Join(tempDir, "audit.log"),
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 512,
		maxLogs:    maxLogs,
	}

	// Rotate well past maxLogs so the oldest slots must be evicted, and label
	// each generation so the ordering can be checked afterwards.
	const rotations = maxLogs + 3
	for i := 1; i <= rotations; i++ {
		if err := os.WriteFile(logger.logPath, []byte(fmt.Sprintf("generation %d\n", i)), 0600); err != nil {
			t.Fatalf("write generation %d: %v", i, err)
		}
		if err := logger.rotateLog(); err != nil {
			t.Fatalf("rotateLog at generation %d: %v", i, err)
		}
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, e := range entries {
		found = append(found, e.Name())
	}

	if len(found) != maxLogs {
		t.Errorf("directory holds %d files, want %d: %v", len(found), maxLogs, found)
	}

	// Slot i must hold the i-th most recent generation, newest in .1.
	for i := 1; i <= maxLogs; i++ {
		path := logger.rotatedLogName(i, true)
		got, err := readGzip(t, path)
		if err != nil {
			t.Errorf("slot %d (%s): %v", i, filepath.Base(path), err)
			continue
		}
		want := fmt.Sprintf("generation %d\n", rotations-i+1)
		if got != want {
			t.Errorf("slot %d holds %q, want %q", i, got, want)
		}
	}

	// Nothing should survive past the retention limit.
	for _, compressed := range []bool{true, false} {
		beyond := logger.rotatedLogName(maxLogs+1, compressed)
		if _, err := os.Stat(beyond); !os.IsNotExist(err) {
			t.Errorf("%s should have been evicted", filepath.Base(beyond))
		}
	}
}

// TestRotationShiftsUncompressedSlots covers the other half: compression is
// best-effort, so a slot can legitimately hold a plain file, and shifting must
// carry that form along too.
func TestRotationShiftsUncompressedSlots(t *testing.T) {
	tempDir := t.TempDir()
	logger := &Logger{
		logPath:    filepath.Join(tempDir, "audit.log"),
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 512,
		maxLogs:    3,
	}

	// Simulate a previous rotation whose compression failed.
	plain := logger.rotatedLogName(1, false)
	if err := os.WriteFile(plain, []byte("uncompressed generation\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(logger.logPath, []byte("current\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := logger.rotateLog(); err != nil {
		t.Fatalf("rotateLog: %v", err)
	}

	moved := logger.rotatedLogName(2, false)
	data, err := os.ReadFile(moved)
	if err != nil {
		t.Fatalf("uncompressed slot was not shifted to %s: %v", filepath.Base(moved), err)
	}
	if string(data) != "uncompressed generation\n" {
		t.Errorf("shifted slot holds %q", data)
	}
}

func readGzip(t *testing.T, path string) (string, error) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer func() { _ = zr.Close() }()
	b, err := io.ReadAll(zr)
	return string(b), err
}
