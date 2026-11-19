package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestScheduleCancellationAcrossProcesses(t *testing.T) {
	setupTestLogger()
	originalDir := pidFileDirOverride
	t.Cleanup(func() { pidFileDirOverride = originalDir })

	if dir := os.Getenv("SYSREBOOT_TEST_SCHEDULE_DIR"); dir != "" {
		pidFileDirOverride = dir
		info := ScheduleInfo{Action: ActionReboot, Delay: 1}
		cleanup, err := setupPIDFile(&info)
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup()

		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		ready, err := waitForScheduleDecision(ctx, time.Minute, info.ScheduleID)
		if err != nil || ready || ctx.Err() != nil {
			t.Fatalf("expected schedule cancellation, got ready=%v, err=%v, context=%v", ready, err, ctx.Err())
		}
		return
	}

	pidFileDirOverride = t.TempDir()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestScheduleCancellationAcrossProcesses$")
	cmd.Env = append(os.Environ(), "SYSREBOOT_TEST_SCHEDULE_DIR="+pidFileDirOverride)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	t.Cleanup(func() {
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})

	for {
		if _, err := readPIDFile(); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("scheduler did not publish its registration")
		case <-time.After(25 * time.Millisecond):
		}
	}
	if err := cancelScheduledAction(); err != nil {
		t.Fatal(err)
	}
	err := cmd.Wait()
	waited = true
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(getPIDFilePath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("scheduler did not remove its registration: %v", err)
	}
}
