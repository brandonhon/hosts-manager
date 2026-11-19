package audit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordAppendsOneLinePerChange(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "audit")
	t.Setenv("SUDO_USER", "alice")

	if err := Record(dir, "add", "Added 10.0.0.5 -> api.test [dev]", nil); err != nil {
		t.Fatal(err)
	}
	if err := Record(dir, "delete", "Removed api.test", errors.New("disk full")); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "changes.log"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d: %q", len(lines), data)
	}
	if !strings.HasSuffix(lines[0], "  alice  add  ok  Added 10.0.0.5 -> api.test [dev]") {
		t.Errorf("line 1 = %q", lines[0])
	}
	if !strings.Contains(lines[1], "  delete  FAILED: disk full  Removed api.test") {
		t.Errorf("line 2 = %q", lines[1])
	}
}

func TestRecordMovesFullLogAside(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "changes.log")
	if err := os.WriteFile(path, make([]byte, maxSize), 0600); err != nil {
		t.Fatal(err)
	}

	if err := Record(dir, "add", "after rotation", nil); err != nil {
		t.Fatal(err)
	}

	if info, err := os.Stat(path + ".1"); err != nil || info.Size() != maxSize {
		t.Fatalf("full log not moved to .1: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "after rotation") || len(data) > 200 {
		t.Errorf("new log should hold only the new line, got %d bytes", len(data))
	}
}
