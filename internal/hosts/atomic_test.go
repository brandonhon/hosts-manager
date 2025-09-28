package hosts

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// Helper function to create test directory
func createTestDir(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "atomic_test_*")
	if err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

// TestNewAtomicFileWriter tests atomic file writer creation
func TestNewAtomicFileWriter(t *testing.T) {
	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tests := []struct {
		name       string
		targetPath string
		expectErr  bool
		setup      func(string) error
	}{
		{
			name:       "new file",
			targetPath: filepath.Join(tmpDir, "test.txt"),
			expectErr:  false,
		},
		{
			name:       "existing file",
			targetPath: filepath.Join(tmpDir, "existing.txt"),
			expectErr:  false,
			setup: func(path string) error {
				return os.WriteFile(path, []byte("existing content"), 0644)
			},
		},
		{
			name:       "file in non-existent directory",
			targetPath: filepath.Join(tmpDir, "nonexistent", "test.txt"),
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				if err := tt.setup(tt.targetPath); err != nil {
					t.Fatal(err)
				}
			}

			writer, err := NewAtomicFileWriter(tt.targetPath)

			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if writer != nil {
				defer func() { _ = writer.Close() }()

				// Verify lock file was created
				lockPath := tt.targetPath + ".lock"
				if _, err := os.Stat(lockPath); os.IsNotExist(err) {
					t.Error("lock file was not created")
				}

				// Verify temp file exists
				if writer.tempFile == nil {
					t.Error("temp file was not created")
				}
			}
		})
	}
}

// TestAtomicFileWriterWrite tests writing to atomic file writer
func TestAtomicFileWriterWrite(t *testing.T) {
	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	targetPath := filepath.Join(tmpDir, "test.txt")
	writer, err := NewAtomicFileWriter(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Close() }()

	tests := []struct {
		name    string
		data    []byte
		expectN int
	}{
		{
			name:    "write simple data",
			data:    []byte("Hello, World!"),
			expectN: 13,
		},
		{
			name:    "write empty data",
			data:    []byte(""),
			expectN: 0,
		},
		{
			name:    "write large data",
			data:    []byte(strings.Repeat("A", 1024)),
			expectN: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := writer.Write(tt.data)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if n != tt.expectN {
				t.Errorf("Write() returned %d, want %d", n, tt.expectN)
			}
		})
	}
}

// TestAtomicFileWriterWriteString tests writing strings to atomic file writer
func TestAtomicFileWriterWriteString(t *testing.T) {
	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	targetPath := filepath.Join(tmpDir, "test.txt")
	writer, err := NewAtomicFileWriter(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Close() }()

	tests := []struct {
		name    string
		str     string
		expectN int
	}{
		{
			name:    "write simple string",
			str:     "Hello, World!",
			expectN: 13,
		},
		{
			name:    "write empty string",
			str:     "",
			expectN: 0,
		},
		{
			name:    "write unicode string",
			str:     "Hello, 世界!",
			expectN: 14, // UTF-8 encoding (世界 = 6 bytes)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := writer.WriteString(tt.str)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if n != tt.expectN {
				t.Errorf("WriteString() returned %d, want %d", n, tt.expectN)
			}
		})
	}
}

// TestAtomicFileWriterCommit tests committing atomic writes
func TestAtomicFileWriterCommit(t *testing.T) {
	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tests := []struct {
		name     string
		content  string
		validate func(*testing.T, string)
	}{
		{
			name:    "commit simple content",
			content: "Hello, World!",
			validate: func(t *testing.T, path string) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("failed to read committed file: %v", err)
				}
				if string(data) != "Hello, World!" {
					t.Errorf("committed content = %q, want %q", string(data), "Hello, World!")
				}
			},
		},
		{
			name:    "commit multi-line content",
			content: "Line 1\nLine 2\nLine 3\n",
			validate: func(t *testing.T, path string) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("failed to read committed file: %v", err)
				}
				lines := strings.Split(string(data), "\n")
				if len(lines) != 4 || lines[0] != "Line 1" {
					t.Errorf("unexpected committed content: %q", string(data))
				}
			},
		},
		{
			name:    "commit empty content",
			content: "",
			validate: func(t *testing.T, path string) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("failed to read committed file: %v", err)
				}
				if len(data) != 0 {
					t.Errorf("expected empty file, got %d bytes", len(data))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new path for each test
			testPath := filepath.Join(tmpDir, fmt.Sprintf("test_%s.txt", tt.name))

			writer, err := NewAtomicFileWriter(testPath)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = writer.Close() }()

			// Write content
			_, err = writer.WriteString(tt.content)
			if err != nil {
				t.Fatalf("failed to write content: %v", err)
			}

			// Commit
			err = writer.Commit()
			if err != nil {
				t.Errorf("Commit() failed: %v", err)
			}

			// Verify file exists
			if _, err := os.Stat(testPath); os.IsNotExist(err) {
				t.Error("committed file does not exist")
			}

			// Verify lock file is cleaned up (after Close() is called)
			_ = writer.Close()
			lockPath := testPath + ".lock"
			if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
				t.Error("lock file was not cleaned up after commit and close")
			}

			if tt.validate != nil {
				tt.validate(t, testPath)
			}
		})
	}
}

// TestAtomicFileWriterClose tests closing the writer
func TestAtomicFileWriterClose(t *testing.T) {
	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	targetPath := filepath.Join(tmpDir, "test.txt")
	writer, err := NewAtomicFileWriter(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	tempPath := writer.tempPath
	lockPath := targetPath + ".lock"

	// Write some content
	_, err = writer.WriteString("test content")
	if err != nil {
		t.Fatal(err)
	}

	// Close without committing
	err = writer.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Verify temp file is cleaned up
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Error("temp file was not cleaned up")
	}

	// Verify lock file is cleaned up
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("lock file was not cleaned up")
	}

	// Verify target file was not created (since we didn't commit)
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Error("target file should not exist without commit")
	}
}

// TestAtomicFileWriterDoubleClose tests closing an already closed writer
func TestAtomicFileWriterDoubleClose(t *testing.T) {
	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	targetPath := filepath.Join(tmpDir, "test.txt")
	writer, err := NewAtomicFileWriter(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	// First close
	err = writer.Close()
	if err != nil {
		t.Errorf("first Close() failed: %v", err)
	}

	// Second close should not panic or cause issues
	err = writer.Close()
	if err != nil {
		t.Errorf("second Close() failed: %v", err)
	}
}

// TestAtomicFileWriterCommitAfterClose tests committing after close
func TestAtomicFileWriterCommitAfterClose(t *testing.T) {
	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	targetPath := filepath.Join(tmpDir, "test.txt")
	writer, err := NewAtomicFileWriter(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	_ = writer.Close()

	// Commit after close should fail
	err = writer.Commit()
	if err == nil {
		t.Error("Commit() after Close() should fail")
	}
}

// TestAtomicFileWriterWriteAfterClose tests writing after close
func TestAtomicFileWriterWriteAfterClose(t *testing.T) {
	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	targetPath := filepath.Join(tmpDir, "test.txt")
	writer, err := NewAtomicFileWriter(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	_ = writer.Close()

	// Write after close should fail
	_, err = writer.Write([]byte("test"))
	if err == nil {
		t.Error("Write() after Close() should fail")
	}

	_, err = writer.WriteString("test")
	if err == nil {
		t.Error("WriteString() after Close() should fail")
	}
}

// TestAtomicWrite tests the convenience function
func TestAtomicWrite(t *testing.T) {
	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tests := []struct {
		name      string
		content   string
		writeFunc func(io.Writer) error
		expectErr bool
	}{
		{
			name:    "successful write",
			content: "Hello, World!",
			writeFunc: func(w io.Writer) error {
				_, err := w.Write([]byte("Hello, World!"))
				return err
			},
			expectErr: false,
		},
		{
			name:    "write function returns error",
			content: "",
			writeFunc: func(w io.Writer) error {
				return fmt.Errorf("simulated write error")
			},
			expectErr: true,
		},
		{
			name:    "multi-step write",
			content: "Line 1\nLine 2\n",
			writeFunc: func(w io.Writer) error {
				if _, err := w.Write([]byte("Line 1\n")); err != nil {
					return err
				}
				if _, err := w.Write([]byte("Line 2\n")); err != nil {
					return err
				}
				return nil
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetPath := filepath.Join(tmpDir, fmt.Sprintf("atomic_%s.txt", tt.name))

			err := AtomicWrite(targetPath, tt.writeFunc)

			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectErr {
				// Verify file was created and has correct content
				data, err := os.ReadFile(targetPath)
				if err != nil {
					t.Errorf("failed to read atomic write result: %v", err)
				}
				if string(data) != tt.content {
					t.Errorf("atomic write result = %q, want %q", string(data), tt.content)
				}

				// Verify no lock file remains
				lockPath := targetPath + ".lock"
				if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
					t.Error("lock file was not cleaned up after atomic write")
				}
			}
		})
	}
}

// TestSafeRead tests safe file reading
func TestSafeRead(t *testing.T) {
	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		filePath  string
		expectErr bool
		expected  string
	}{
		{
			name:      "read existing file",
			filePath:  testFile,
			expectErr: false,
			expected:  testContent,
		},
		{
			name:      "read non-existent file",
			filePath:  filepath.Join(tmpDir, "nonexistent.txt"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := SafeRead(tt.filePath)

			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectErr && string(data) != tt.expected {
				t.Errorf("SafeRead() = %q, want %q", string(data), tt.expected)
			}
		})
	}
}

// TestIsFileLocked tests file lock detection
func TestIsFileLocked(t *testing.T) {
	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	testFile := filepath.Join(tmpDir, "test.txt")

	// Test unlocked file
	locked := IsFileLocked(testFile)
	if locked {
		t.Error("file should not be locked initially")
	}

	// Create lock
	writer, err := NewAtomicFileWriter(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Close() }()

	// Test locked file
	locked = IsFileLocked(testFile)
	if !locked {
		t.Error("file should be locked when writer is active")
	}

	// Close writer
	_ = writer.Close()

	// Test unlocked file after close
	locked = IsFileLocked(testFile)
	if locked {
		t.Error("file should not be locked after writer is closed")
	}
}

// TestConcurrentAtomicWrites tests concurrent atomic write operations
func TestConcurrentAtomicWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	testFile := filepath.Join(tmpDir, "concurrent.txt")

	var wg sync.WaitGroup
	errors := make(chan error, 5)
	successCount := make(chan int, 5)

	// Launch 5 concurrent atomic writes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			content := fmt.Sprintf("Content from goroutine %d\n", id)
			err := AtomicWrite(testFile, func(w io.Writer) error {
				_, err := w.Write([]byte(content))
				// Add small delay to increase chance of contention
				time.Sleep(10 * time.Millisecond)
				return err
			})

			if err != nil {
				errors <- err
			} else {
				successCount <- 1
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	close(successCount)

	// Count successes and errors
	successTotal := 0
	for range successCount {
		successTotal++
	}

	errorTotal := 0
	for err := range errors {
		errorTotal++
		t.Logf("Concurrent write error: %v", err)
	}

	// At least one should succeed due to locking
	if successTotal == 0 {
		t.Error("expected at least one concurrent write to succeed")
	}

	// Total should equal number of goroutines
	if successTotal+errorTotal != 5 {
		t.Errorf("expected 5 total operations, got %d successes + %d errors = %d",
			successTotal, errorTotal, successTotal+errorTotal)
	}

	// Verify final file exists and has some content
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("final file should exist after concurrent writes")
	} else {
		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("failed to read final file: %v", err)
		} else if len(data) == 0 {
			t.Error("final file should have content")
		}
	}
}

// TestStaleLockCleanup tests stale lock file cleanup
func TestStaleLockCleanup(t *testing.T) {
	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	testFile := filepath.Join(tmpDir, "test.txt")
	lockFile := testFile + ".lock"

	// Create a stale lock file with old timestamp
	err := os.WriteFile(lockFile, []byte("old lock"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Make it look old by modifying the timestamp
	oldTime := time.Now().Add(-10 * time.Minute)
	err = os.Chtimes(lockFile, oldTime, oldTime)
	if err != nil {
		t.Fatal(err)
	}

	// Try to create new atomic writer - should clean up stale lock
	writer, err := NewAtomicFileWriter(testFile)
	if err != nil {
		t.Errorf("should be able to clean up stale lock: %v", err)
	}

	if writer != nil {
		defer func() { _ = writer.Close() }()

		// Verify new lock file was created
		if _, err := os.Stat(lockFile); os.IsNotExist(err) {
			t.Error("new lock file should exist")
		}
	}
}

// TestAtomicWritePreservesPermissions tests that atomic writes preserve file permissions
func TestAtomicWritePreservesPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := createTestDir(t)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	testFile := filepath.Join(tmpDir, "test.txt")

	// Create initial file with specific permissions
	initialContent := "initial content"
	err := os.WriteFile(testFile, []byte(initialContent), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Get initial permissions
	stat, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}
	initialMode := stat.Mode()

	// Perform atomic write
	newContent := "new content"
	err = AtomicWrite(testFile, func(w io.Writer) error {
		_, err := w.Write([]byte(newContent))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	// Check that permissions were preserved
	stat, err = os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}
	newMode := stat.Mode()

	if newMode != initialMode {
		t.Errorf("permissions not preserved: initial=%v, final=%v", initialMode, newMode)
	}

	// Verify content was updated
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != newContent {
		t.Errorf("content not updated: got %q, want %q", string(data), newContent)
	}
}

// Benchmark tests
func BenchmarkAtomicWrite(b *testing.B) {
	tmpDir := createTestDirB(b)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	content := strings.Repeat("Hello, World!\n", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testFile := filepath.Join(tmpDir, fmt.Sprintf("bench_%d.txt", i))
		err := AtomicWrite(testFile, func(w io.Writer) error {
			_, err := w.Write([]byte(content))
			return err
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAtomicWriteSmallFile(b *testing.B) {
	tmpDir := createTestDirB(b)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	content := "Hello, World!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testFile := filepath.Join(tmpDir, fmt.Sprintf("small_%d.txt", i))
		err := AtomicWrite(testFile, func(w io.Writer) error {
			_, err := w.Write([]byte(content))
			return err
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSafeRead(b *testing.B) {
	tmpDir := createTestDirB(b)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	content := strings.Repeat("Hello, World!\n", 100)
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := SafeRead(testFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper for benchmarks
func createTestDirB(b *testing.B) string {
	b.Helper()
	tmpDir, err := os.MkdirTemp("", "atomic_test_*")
	if err != nil {
		b.Fatal(err)
	}
	return tmpDir
}
