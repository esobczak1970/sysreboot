package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// Cancellation and execution tests exercise the same filesystem decision
// protocol used in production. No real process is signaled.
func registerTestSchedule(t *testing.T) string {
	t.Helper()
	setupTestLogger()
	oldDir := pidFileDirOverride
	pidFileDirOverride = t.TempDir()
	t.Cleanup(func() { pidFileDirOverride = oldDir })
	info := ScheduleInfo{Action: ActionReboot}
	cleanup, err := setupPIDFile(&info)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	return info.ScheduleID
}

func TestCancelScheduledAction_ExecutionAlreadyClaimed(t *testing.T) {
	id := registerTestSchedule(t)
	originalLookup := findProcessFunc
	findProcessFunc = func(int) (processController, error) {
		t.Fatal("cancellation must never open a process handle")
		return nil, nil
	}
	t.Cleanup(func() { findProcessFunc = originalLookup })
	won, err := claimRegisteredSchedule(id)
	if !won || err != nil {
		t.Fatalf("execution claim failed: %v %v", won, err)
	}
	if err := cancelScheduledAction(); err == nil {
		t.Fatal("cannot confirm cancellation once execution has claimed the decision")
	}
	if _, err := os.Stat(getPIDFilePath()); err != nil {
		t.Fatal("execution registration must remain:", err)
	}
}

func TestCancelScheduledAction_StaleRegistrationCannotClaimReplacement(t *testing.T) {
	id := registerTestSchedule(t)
	if err := removePIDFile(); err != nil {
		t.Fatal(err)
	}
	replacement := ScheduleInfo{Action: ActionPoweroff}
	cleanup, err := setupPIDFile(&replacement)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if won, err := claimRegisteredSchedule(id); won || err == nil {
		t.Fatalf("old identity must not claim replacement: %v %v", won, err)
	}
	if decided, err := scheduleDecisionExists(replacement.ScheduleID); decided || err != nil {
		t.Fatalf("replacement unexpectedly decided: %v %v", decided, err)
	}
	cleanupScheduleRegistration(id)
	info, err := readPIDFile()
	if err != nil || info.ScheduleID != replacement.ScheduleID {
		t.Fatalf("stale cleanup changed new registration: %v %v", info, err)
	}
}

func TestInvalidScheduledPIDsAreNeverSignaled(t *testing.T) {
	for _, pid := range []int{-2, -1, 0, 1} {
		t.Run(fmt.Sprintf("pid_%d", pid), func(t *testing.T) {
			setupTestLogger()
			oldDir := pidFileDirOverride
			pidFileDirOverride = t.TempDir()
			t.Cleanup(func() { pidFileDirOverride = oldDir })

			data := []byte(fmt.Sprintf(`{"pid":%d,"action":"reboot"}`, pid))
			if err := os.WriteFile(getPIDFilePath(), data, 0o600); err != nil {
				t.Fatal(err)
			}

			originalLookup := findProcessFunc
			calls := 0
			findProcessFunc = func(int) (processController, error) {
				calls++
				return nil, errors.New("unexpected process lookup")
			}
			t.Cleanup(func() { findProcessFunc = originalLookup })

			if err := cancelScheduledAction(); err == nil || !strings.Contains(err.Error(), "invalid scheduled process PID") {
				t.Fatalf("cancelScheduledAction() error = %v; want invalid PID", err)
			}
			for _, platform := range []string{"linux", "windows"} {
				if processExistsInternal(pid, platform) {
					t.Fatalf("processExistsInternal(%d, %q) unexpectedly true", pid, platform)
				}
			}
			if calls != 0 {
				t.Fatalf("invalid PID %d resulted in %d process lookups", pid, calls)
			}
			if _, err := os.Stat(getPIDFilePath()); err != nil {
				t.Fatalf("invalid schedule file unexpectedly removed: %v", err)
			}
		})
	}
}

func TestWaitForScheduleDecisionStopsPendingTimer(t *testing.T) {
	id := registerTestSchedule(t)
	if err := cancelScheduledAction(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ready, err := waitForScheduleDecision(ctx, time.Hour, id)
	if ready || err != nil || ctx.Err() != nil {
		t.Fatalf("pending timer did not observe cancellation: ready=%v err=%v context=%v", ready, err, ctx.Err())
	}
}

func TestWaitForScheduleDecisionMissingRegistrationFailsClosed(t *testing.T) {
	id := registerTestSchedule(t)
	if err := removePIDFile(); err != nil {
		t.Fatal(err)
	}
	ready, err := waitForScheduleDecision(context.Background(), 0, id)
	if ready || err == nil {
		t.Fatalf("missing registration allowed execution: ready=%v err=%v", ready, err)
	}
}

func TestWaitForScheduleDecisionCanExecuteOnlyOnce(t *testing.T) {
	id := registerTestSchedule(t)
	ready, err := waitForScheduleDecision(context.Background(), 0, id)
	if !ready || err != nil {
		t.Fatalf("valid registration did not win: ready=%v err=%v", ready, err)
	}
	ready, err = waitForScheduleDecision(context.Background(), 0, id)
	if ready || err != nil {
		t.Fatalf("duplicate execution was allowed: ready=%v err=%v", ready, err)
	}
}

func TestScheduleStatusPreservesMalformedRegistration(t *testing.T) {
	setupTestLogger()
	oldDir := pidFileDirOverride
	pidFileDirOverride = t.TempDir()
	t.Cleanup(func() { pidFileDirOverride = oldDir })
	const invalid = "{broken registration"
	if err := os.WriteFile(getPIDFilePath(), []byte(invalid), 0600); err != nil {
		t.Fatal(err)
	}
	if err := showScheduleStatus(); err == nil {
		t.Fatal("corrupt registration must not be reported as no schedule")
	}
	got, err := os.ReadFile(getPIDFilePath())
	if err != nil || string(got) != invalid {
		t.Fatalf("status changed corrupt registration: %q, %v", got, err)
	}
}
