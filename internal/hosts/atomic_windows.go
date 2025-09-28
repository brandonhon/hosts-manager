//go:build windows

package hosts

import (
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

const (
	LOCKFILE_EXCLUSIVE_LOCK   = 0x00000002
	LOCKFILE_FAIL_IMMEDIATELY = 0x00000001
)

// platformAcquireLock acquires an exclusive lock on the file
func platformAcquireLock(fd int) error {
	handle := syscall.Handle(fd)
	var overlapped syscall.Overlapped

	ret, _, err := procLockFileEx.Call(
		uintptr(handle),
		uintptr(LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY),
		uintptr(0),
		uintptr(0xFFFFFFFF),
		uintptr(0xFFFFFFFF),
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if ret == 0 {
		return err
	}
	return nil
}

// platformReleaseLock releases the lock on the file
func platformReleaseLock(fd int) error {
	handle := syscall.Handle(fd)
	var overlapped syscall.Overlapped

	ret, _, err := procUnlockFileEx.Call(
		uintptr(handle),
		uintptr(0),
		uintptr(0xFFFFFFFF),
		uintptr(0xFFFFFFFF),
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if ret == 0 {
		return err
	}
	return nil
}

// platformAcquireSharedLock acquires a shared lock on the file
func platformAcquireSharedLock(fd int) error {
	handle := syscall.Handle(fd)
	var overlapped syscall.Overlapped

	ret, _, err := procLockFileEx.Call(
		uintptr(handle),
		uintptr(LOCKFILE_FAIL_IMMEDIATELY), // No exclusive flag = shared lock
		uintptr(0),
		uintptr(0xFFFFFFFF),
		uintptr(0xFFFFFFFF),
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if ret == 0 {
		return err
	}
	return nil
}
