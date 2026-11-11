package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestCancellationRejectsInvalidPIDsWithoutSignaling(t *testing.T) {
	for _, pid := range []int{-2, -1, 0, 1} {
		t.Run(strconv.Itoa(pid), func(t *testing.T) {
			setupTestLogger()
			originalDir := pidFileDirOverride
			pidFileDirOverride = t.TempDir()
			t.Cleanup(func() { pidFileDirOverride = originalDir })

			payload, err := json.Marshal(ScheduleInfo{PID: pid, Action: ActionReboot})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(pidFileDirOverride, pidFileName), payload, 0600); err != nil {
				t.Fatal(err)
			}

			originalFind := findProcessFunc
			calls := 0
			findProcessFunc = func(pid int) (processController, error) {
				calls++
				return nil, nil
			}
			t.Cleanup(func() { findProcessFunc = originalFind })

			if err := cancelScheduledAction(); err == nil || !strings.Contains(err.Error(), "invalid scheduled process PID") {
				t.Fatalf("cancelScheduledAction() error = %v; want invalid PID", err)
			}
			if calls != 0 {
				t.Fatalf("unexpected process lookup/signaling for PID %d: %d", pid, calls)
			}
			if _, err := os.Stat(getPIDFilePath()); err != nil {
				t.Fatalf("invalid PID file was removed: %v", err)
			}
		})
	}
}

func TestProcessExistsRejectsInvalidPIDsWithoutLookup(t *testing.T) {
	originalFind := findProcessFunc
	calls := 0
	findProcessFunc = func(pid int) (processController, error) {
		calls++
		return nil, nil
	}
	t.Cleanup(func() { findProcessFunc = originalFind })

	for _, pid := range []int{-1, 0, 1} {
		if processExistsInternal(pid, "linux") || processExistsInternal(pid, "windows") {
			t.Errorf("processExistsInternal(%d) unexpectedly reported an active process", pid)
		}
	}
	if calls != 0 {
		t.Fatalf("unexpected process lookups for invalid PIDs: %d", calls)
	}
}
