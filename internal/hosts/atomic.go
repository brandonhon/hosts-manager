package hosts

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// AtomicFileWriter provides atomic file writing with locking
type AtomicFileWriter struct {
	targetPath string
	tempPath   string
	lockFile   *os.File
	tempFile   *os.File
}

// NewAtomicFileWriter creates a new atomic file writer
func NewAtomicFileWriter(targetPath string) (*AtomicFileWriter, error) {
	// Create temporary file in the same directory to ensure atomic rename
	dir := filepath.Dir(targetPath)
	lockPath := targetPath + ".lock"

	// Create lock file first with O_EXCL for atomic creation
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			// Check if lock file is stale (older than 5 minutes)
			if info, statErr := os.Stat(lockPath); statErr == nil {
				age := time.Since(info.ModTime())
				if age > 5*time.Minute {
					// Attempt to clean up stale lock file
					if rmErr := os.Remove(lockPath); rmErr == nil {
						// Retry lock file creation
						lockFile, err = os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
						if err != nil {
							return nil, fmt.Errorf("failed to create lock file after cleanup: %w", err)
						}
					} else {
						return nil, fmt.Errorf("file is locked by another process (stale lock cleanup failed): %s", targetPath)
					}
				} else {
					return nil, fmt.Errorf("file is locked by another process: %s", targetPath)
				}
			} else {
				return nil, fmt.Errorf("file is locked by another process: %s", targetPath)
			}
		} else {
			return nil, fmt.Errorf("failed to create lock file: %w", err)
		}
	}

	// Write PID and timestamp to lock file for debugging and stale detection
	lockInfo := fmt.Sprintf("%d\n%s\n", os.Getpid(), time.Now().Format(time.RFC3339))
	if _, err := lockFile.WriteString(lockInfo); err != nil {
		lockFile.Close()
		os.Remove(lockPath)
		return nil, fmt.Errorf("failed to write lock info to lock file: %w", err)
	}

	// Acquire exclusive lock
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lockFile.Close()
		os.Remove(lockPath)
		return nil, fmt.Errorf("failed to acquire file lock: %w", err)
	}

	// Get original file permissions
	var fileMode os.FileMode = 0644
	if stat, err := os.Stat(targetPath); err == nil {
		fileMode = stat.Mode()
	}

	// Create secure temporary file using os.CreateTemp in the same directory
	tempFile, err := os.CreateTemp(dir, "."+filepath.Base(targetPath)+".tmp.*")
	if err != nil {
		syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		lockFile.Close()
		os.Remove(lockPath)
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}

	// Set appropriate permissions on the temporary file
	if err := tempFile.Chmod(fileMode); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		lockFile.Close()
		os.Remove(lockPath)
		return nil, fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	return &AtomicFileWriter{
		targetPath: targetPath,
		tempPath:   tempFile.Name(), // Use the actual secure temporary file name
		lockFile:   lockFile,
		tempFile:   tempFile,
	}, nil
}

// Write writes data to the temporary file
func (aw *AtomicFileWriter) Write(data []byte) (int, error) {
	if aw.tempFile == nil {
		return 0, fmt.Errorf("writer has been closed")
	}
	return aw.tempFile.Write(data)
}

// WriteString writes a string to the temporary file
func (aw *AtomicFileWriter) WriteString(s string) (int, error) {
	if aw.tempFile == nil {
		return 0, fmt.Errorf("writer has been closed")
	}
	return aw.tempFile.WriteString(s)
}

// Commit atomically moves the temporary file to the target location
func (aw *AtomicFileWriter) Commit() error {
	if aw.tempFile == nil {
		return fmt.Errorf("writer has been closed")
	}

	// Flush and sync the temporary file
	if err := aw.tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	// Close the temporary file
	if err := aw.tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	aw.tempFile = nil

	// Atomic rename
	if err := os.Rename(aw.tempPath, aw.targetPath); err != nil {
		return fmt.Errorf("failed to commit file: %w", err)
	}

	return nil
}

// Close cleans up resources and releases the lock
func (aw *AtomicFileWriter) Close() error {
	var lastErr error

	// Close temporary file if still open
	if aw.tempFile != nil {
		if err := aw.tempFile.Close(); err != nil {
			lastErr = err
		}
		aw.tempFile = nil
	}

	// Remove temporary file if it exists
	if aw.tempPath != "" {
		if err := os.Remove(aw.tempPath); err != nil && !os.IsNotExist(err) {
			lastErr = err
		}
	}

	// Release file lock and close lock file
	if aw.lockFile != nil {
		syscall.Flock(int(aw.lockFile.Fd()), syscall.LOCK_UN)
		if err := aw.lockFile.Close(); err != nil {
			lastErr = err
		}

		// Remove lock file
		lockPath := aw.targetPath + ".lock"
		if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
			lastErr = err
		}
		aw.lockFile = nil
	}

	return lastErr
}

// AtomicWrite performs an atomic write operation with a callback
func AtomicWrite(targetPath string, writeFunc func(io.Writer) error) error {
	writer, err := NewAtomicFileWriter(targetPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	// Write data using the provided function
	if err := writeFunc(writer); err != nil {
		return fmt.Errorf("write operation failed: %w", err)
	}

	// Commit the changes
	if err := writer.Commit(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// SafeRead reads a file with a shared lock to prevent reading during writes
func SafeRead(filePath string) ([]byte, error) {
	lockPath := filePath + ".lock"

	// Check if lock file exists (indicating a write in progress)
	if _, err := os.Stat(lockPath); err == nil {
		// Wait a bit for the write to complete
		time.Sleep(100 * time.Millisecond)

		// Check again
		if _, err := os.Stat(lockPath); err == nil {
			return nil, fmt.Errorf("file is currently being written to: %s", filePath)
		}
	}

	// Open file with shared lock
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Acquire shared lock
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err != nil {
		return nil, fmt.Errorf("failed to acquire shared lock: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	// Read the file
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// IsFileLocked checks if a file is currently locked
func IsFileLocked(filePath string) bool {
	lockPath := filePath + ".lock"
	_, err := os.Stat(lockPath)
	return err == nil
}
