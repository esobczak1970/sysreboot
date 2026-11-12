package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A scheduled reboot must never continue if a separate CLI invocation
// cannot discover and cancel it through the registered PID file.
func TestSetupPIDFileReturnsErrorWithoutCleanupWhenRegistrationFails(t *testing.T) {
	setupTestLogger()
	originalDir := pidFileDirOverride
	pidFileDirOverride = filepath.Join(t.TempDir(), "missing-directory")
	t.Cleanup(func() { pidFileDirOverride = originalDir })

	cleanup, err := setupPIDFile(ScheduleInfo{Action: ActionReboot})
	if err == nil || !strings.Contains(err.Error(), "unable to register scheduled action for cancellation") {
		t.Fatalf("expected PID registration error, got %v", err)
	}
	if cleanup != nil {
		t.Fatal("failed PID registration returned a cleanup that could remove an unrelated file")
	}
}

func TestSetupPIDFileRegistersAndCleansUpSuccessfulSchedule(t *testing.T) {
	setupTestLogger()
	originalDir := pidFileDirOverride
	pidFileDirOverride = t.TempDir()
	t.Cleanup(func() { pidFileDirOverride = originalDir })

	cleanup, err := setupPIDFile(ScheduleInfo{Action: ActionReboot})
	if err != nil {
		t.Fatalf("setupPIDFile() failed: %v", err)
	}
	if cleanup == nil {
		t.Fatal("successful registration did not provide cleanup")
	}
	info, err := readPIDFile()
	if err != nil {
		t.Fatalf("registered schedule should be readable: %v", err)
	}
	if info.PID != os.Getpid() || info.Action != ActionReboot {
		t.Fatalf("unexpected schedule information: %+v", info)
	}
	cleanup()
	if _, err := os.Stat(getPIDFilePath()); !os.IsNotExist(err) {
		t.Fatalf("successful cleanup should remove PID file, got %v", err)
	}
}

func TestHandleScheduledTimeFailsClosedIfPIDFileCannotBeWritten(t *testing.T) {
	setupTestLogger()
	resetFlags()

	oldDir := pidFileDirOverride
	pidFileDirOverride = filepath.Join(t.TempDir(), "missing-directory")
	t.Cleanup(func() { pidFileDirOverride = oldDir })
	oldExit := osExit
	exitCode := -1
	osExit = func(code int) { exitCode = code }
	t.Cleanup(func() { osExit = oldExit })

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Prevent a buggy regression from waiting for a real schedule.
	handleScheduledTime(ctx, time.Now().Add(time.Hour).Format("15:04"), ActionReboot)
	if exitCode != 1 {
		t.Fatalf("uncancellable scheduled action should exit with 1, got %d", exitCode)
	}
}

func TestHandleDelayFailsClosedIfPIDFileCannotBeWritten(t *testing.T) {
	setupTestLogger()
	resetFlags()

	oldDir := pidFileDirOverride
	pidFileDirOverride = filepath.Join(t.TempDir(), "missing-directory")
	t.Cleanup(func() { pidFileDirOverride = oldDir })
	oldExit := osExit
	exitCode := -1
	osExit = func(code int) { exitCode = code }
	t.Cleanup(func() { osExit = oldExit })

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Ensure tests can never run a real delayed system action.
	handleDelay(ctx, 5, ActionReboot)
	if exitCode != 1 {
		t.Fatalf("uncancellable delayed action should exit with 1, got %d", exitCode)
	}
}
