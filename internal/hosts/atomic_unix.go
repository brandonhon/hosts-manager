//go:build unix || linux || darwin

package hosts

import (
	"syscall"
)

// platformAcquireLock acquires an exclusive lock on the file
func platformAcquireLock(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
}

// platformReleaseLock releases the lock on the file
func platformReleaseLock(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_UN)
}

// platformAcquireSharedLock acquires a shared lock on the file
func platformAcquireSharedLock(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_SH|syscall.LOCK_NB)
}