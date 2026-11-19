// Package audit keeps a plain-text record of changes made to the hosts file,
// one line per change.
package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// maxSize is the size at which the log is moved aside to changes.log.1,
// replacing any previous one. At roughly 100 bytes a line that is about ten
// thousand changes, so the two files together hold a long history.
const maxSize = 1 << 20

// Record appends one line describing a change to dir/changes.log:
//
//	2026-11-19T15:04:05Z  alice  add  ok  Added 10.0.0.5 -> api.test [dev]
//
// The user is the one who ran sudo, when sudo was used.
func Record(dir, op, summary string, opErr error) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, "changes.log")

	if info, err := os.Stat(path); err == nil && info.Size() >= maxSize {
		if err := os.Rename(path, path+".1"); err != nil {
			return err
		}
	}

	result := "ok"
	if opErr != nil {
		result = "FAILED: " + opErr.Error()
	}
	line := fmt.Sprintf("%s  %s  %s  %s  %s\n",
		time.Now().UTC().Format(time.RFC3339), user(), op, result, summary)

	// #nosec G304 -- fixed filename under the tool's own data directory
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(line); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func user() string {
	for _, key := range []string{"SUDO_USER", "USER", "USERNAME"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return "unknown"
}
