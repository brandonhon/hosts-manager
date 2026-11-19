//go:build !windows

package platform

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
)

// Run as root via sudo to exercise this, e.g.
//
//	go test -c -o /tmp/platform.test ./internal/platform && sudo /tmp/platform.test -test.run UnderSudo -test.v
func TestOwnershipHelpersUnderSudo(t *testing.T) {
	if os.Geteuid() != 0 || os.Getenv("SUDO_USER") == "" {
		t.Skip("needs to run as root under sudo")
	}
	u, err := user.Lookup(os.Getenv("SUDO_USER"))
	if err != nil {
		t.Fatal(err)
	}
	wantUID, _ := strconv.Atoi(u.Uid)

	base, err := os.MkdirTemp(u.HomeDir, ".hosts-manager-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })

	owner := func(path string) int {
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		return int(info.Sys().(*syscall.Stat_t).Uid)
	}

	dir := filepath.Join(base, "data", "audit")
	if err := MkdirAllForUser(dir, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "changes.log")
	if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := GiveToInvokingUser(file); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(base, "data"), dir, file} {
		if got := owner(p); got != wantUID {
			t.Errorf("%s owned by uid %d, want %d", p, got, wantUID)
		}
	}

	// Outside the user's home nothing is handed over.
	outside, err := os.MkdirTemp("", "hosts-manager-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(outside) })
	if err := GiveToInvokingUser(outside); err != nil {
		t.Fatal(err)
	}
	if got := owner(outside); got != 0 {
		t.Errorf("%s outside home given to uid %d, want root", outside, got)
	}
}
