package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

type cancellationTestProcess struct {
	exists       bool
	interruptErr error
	killErr      error
	signals      []os.Signal
	killCalls    int
}

func (p *cancellationTestProcess) Signal(signal os.Signal) error {
	p.signals = append(p.signals, signal)
	if signal == syscall.Signal(0) {
		if !p.exists {
			return errors.New("process no longer exists")
		}
		return nil
	}
	if signal == os.Interrupt {
		return p.interruptErr
	}
	return fmt.Errorf("unexpected signal: %v", signal)
}

func (p *cancellationTestProcess) Kill() error {
	p.killCalls++
	return p.killErr
}

func TestCancelScheduledAction_ProcessFailures(t *testing.T) {
	cases := []struct {
		name          string
		exists        bool
		interruptErr  error
		killErr       error
		lookupErr     error
		wantError     string
		wantPIDFile   bool
		wantKillCalls int
		wantLookups   int
	}{
		{
			name:        "interrupt success removes PID file",
			exists:      true,
			wantLookups: 2,
		},
		{
			name:          "failed interrupt falls back to kill",
			exists:        true,
			interruptErr:  errors.New("interrupt failed"),
			wantKillCalls: 1,
			wantLookups:   2,
		},
		{
			name:          "failed kill preserves PID file",
			exists:        true,
			interruptErr:  errors.New("interrupt failed"),
			killErr:       errors.New("kill failed"),
			wantError:     "failed to kill process: kill failed",
			wantPIDFile:   true,
			wantKillCalls: 1,
			wantLookups:   2,
		},
		{
			name:        "second process lookup fails without deleting PID file",
			exists:      true,
			lookupErr:   errors.New("lookup failed"),
			wantError:   "failed to find process: lookup failed",
			wantPIDFile: true,
			wantLookups: 2,
		},
		{
			name:        "stale PID file is removed",
			exists:      false,
			wantError:   "no active scheduled action found",
			wantLookups: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if runtime.GOOS == "windows" && !tc.exists {
				t.Skip("Windows processExists does not use Unix signal 0")
			}

			setupTestLogger()

			originalPIDOverride := pidFileDirOverride
			pidFileDirOverride = t.TempDir()
			originalLookup := findProcessFunc
			t.Cleanup(func() {
				findProcessFunc = originalLookup
				pidFileDirOverride = originalPIDOverride
			})

			// writePIDFile records the current test PID, but every lookup in this
			// test uses the mock; no real process is signaled or terminated.
			if err := writePIDFile(ScheduleInfo{Action: ActionReboot}); err != nil {
				t.Fatalf("writePIDFile: %v", err)
			}

			process := &cancellationTestProcess{
				exists:       tc.exists,
				interruptErr: tc.interruptErr,
				killErr:      tc.killErr,
			}
			lookups := 0
			findProcessFunc = func(pid int) (processController, error) {
				if pid != os.Getpid() {
					return nil, fmt.Errorf("unexpected PID %d", pid)
				}
				lookups++
				if lookups == 2 && tc.lookupErr != nil {
					return nil, tc.lookupErr
				}
				return process, nil
			}

			err := cancelScheduledAction()
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("cancelScheduledAction: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("cancelScheduledAction error = %v, want %q", err, tc.wantError)
			}

			if lookups != tc.wantLookups {
				t.Errorf("process lookup count = %d, want %d", lookups, tc.wantLookups)
			}
			if process.killCalls != tc.wantKillCalls {
				t.Errorf("Kill call count = %d, want %d", process.killCalls, tc.wantKillCalls)
			}

			// Verify the same signal sequence used in production: an existence
			// probe on Unix, then SIGINT only when the second lookup succeeds.
			wantSignals := 0
			if runtime.GOOS != "windows" {
				wantSignals++
				if len(process.signals) > 0 && process.signals[0] != syscall.Signal(0) {
					t.Errorf("first signal = %v, want signal 0", process.signals[0])
				}
			}
			if tc.exists && tc.lookupErr == nil {
				wantSignals++
				if len(process.signals) > 0 && process.signals[len(process.signals)-1] != os.Interrupt {
					t.Errorf("last signal = %v, want interrupt", process.signals[len(process.signals)-1])
				}
			}
			if got := len(process.signals); got != wantSignals {
				t.Errorf("signal count = %d, want %d", got, wantSignals)
			}

			_, statErr := os.Stat(getPIDFilePath())
			if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("stat PID file: %v", statErr)
			}
			if got := statErr == nil; got != tc.wantPIDFile {
				t.Errorf("PID file exists = %t, want %t", got, tc.wantPIDFile)
			}
		})
	}
}
