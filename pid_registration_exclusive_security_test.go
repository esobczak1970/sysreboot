package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestScheduledPIDFileCannotOverwriteActiveRegistration(t *testing.T) {
	setupTestLogger()
	oldDir := pidFileDirOverride
	pidFileDirOverride = t.TempDir()
	t.Cleanup(func() { pidFileDirOverride = oldDir })
	if err := writePIDFile(ScheduleInfo{Action: ActionReboot, Delay: 10}); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(getPIDFilePath())
	if err != nil {
		t.Fatal(err)
	}
	if err := writePIDFile(ScheduleInfo{Action: ActionPoweroff, Delay: 5}); !errors.Is(err, os.ErrExist) {
		t.Fatalf("second registration err=%v, want os.ErrExist", err)
	}
	got, err := os.ReadFile(getPIDFilePath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, got) {
		t.Fatalf("second registration replaced cancellation state: before=%s after=%s", original, got)
	}
	// Explicit removal is the recovery path for a confirmed stale record.
	if err := removePIDFile(); err != nil {
		t.Fatal(err)
	}
	if err := writePIDFile(ScheduleInfo{Action: ActionPoweroff, Delay: 5}); err != nil {
		t.Fatalf("registration should work after explicit removal: %v", err)
	}
	info, err := readPIDFile()
	if err != nil || info.Action != ActionPoweroff {
		t.Fatalf("new registration info=%+v err=%v", info, err)
	}
}

func TestScheduledPIDFileDoesNotRemovePreexistingSymlink(t *testing.T) {
	setupTestLogger()
	oldDir := pidFileDirOverride
	pidFileDirOverride = t.TempDir()
	t.Cleanup(func() { pidFileDirOverride = oldDir })
	victim := filepath.Join(t.TempDir(), "untouched.txt")
	if err := os.WriteFile(victim, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	pidPath := getPIDFilePath()
	if err := os.Symlink(victim, pidPath); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	err := writePIDFile(ScheduleInfo{Action: ActionReboot})
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("preexisting link should make registration fail closed, got %v", err)
	}
	link, err := os.Readlink(pidPath)
	if err != nil || link != victim {
		t.Fatalf("existing link was removed or replaced: %q, %v", link, err)
	}
	bytesRead, err := os.ReadFile(victim)
	if err != nil || string(bytesRead) != "untouched" {
		t.Fatalf("external file modified: %q, %v", bytesRead, err)
	}
}
