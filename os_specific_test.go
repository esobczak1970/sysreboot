package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestExecuteSystemCommandInternal_OS(t *testing.T) {
	setupTestLogger()

	oldExecCmdFunc := execCmdFunc
	defer func() { execCmdFunc = oldExecCmdFunc }()

	var lastCmdName string
	var lastCmdArgs []string

	execCmdFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		lastCmdName = name
		lastCmdArgs = args
		return harmlessTestCommand(ctx, "success")
	}

	tests := []struct {
		name     string
		goos     string
		action   string
		wantCmd  string
		wantArgs []string
		wantErr  bool
	}{
		{"Linux Reboot", "linux", ActionReboot, "systemctl", []string{"reboot"}, false},
		{"Linux Poweroff", "linux", ActionPoweroff, "systemctl", []string{"poweroff"}, false},
		{"Linux Halt", "linux", ActionHalt, "systemctl", []string{"halt"}, false},
		{"Windows Reboot", "windows", ActionReboot, "shutdown.exe", []string{"/r", "/t", "0"}, false},
		{"Windows Poweroff", "windows", ActionPoweroff, "shutdown.exe", []string{"/s", "/t", "0"}, false},
		{"Windows Halt", "windows", ActionHalt, "shutdown.exe", []string{"/s", "/t", "0"}, false},
		{"Windows Invalid", "windows", "invalid", "", nil, true},
		{"macOS Reboot", "darwin", ActionReboot, "shutdown", []string{"-r", "now"}, false},
		{"macOS Poweroff", "darwin", ActionPoweroff, "shutdown", []string{"-h", "now"}, false},
		{"macOS Halt", "darwin", ActionHalt, "halt", []string{}, false},
		{"macOS Invalid", "darwin", "invalid", "", nil, true},
		{"Unsupported OS", "unsupported", ActionReboot, "", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lastCmdName = ""
			lastCmdArgs = nil

			err := executeSystemCommandInternal(tt.action, tt.goos)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for %s %s, got nil", tt.goos, tt.action)
				}
				// Skip arg checking if it's supposed to fail early
				if tt.goos == "darwin" && tt.action != "invalid" && err != nil && strings.Contains(err.Error(), "root privileges required") {
					_ = err
					// This is expected when running Darwin test on non-root (or if we mock the root check).
					// In executeSystemCommandInternal it checks os.Geteuid() != 0.
					// If we run this on a non-root environment, it will fail for darwin.
				}
			} else {
				if tt.goos == "darwin" {
					// We might get root privilege error, skip arg checking.
					if err != nil && strings.Contains(err.Error(), "root privileges required") {
						return
					}
				}
				if err != nil {
					// Expected when running windows mock on non-windows.
					if tt.goos == "windows" && strings.Contains(err.Error(), "windows system directory is unavailable") {
						return
					}
					t.Errorf("Unexpected error for %s %s: %v", tt.goos, tt.action, err)
				}

				if !strings.HasSuffix(lastCmdName, tt.wantCmd) {
					t.Errorf("Expected cmd %s, got %s", tt.wantCmd, lastCmdName)
				}

				if len(lastCmdArgs) != len(tt.wantArgs) {
					t.Errorf("Expected args %v, got %v", tt.wantArgs, lastCmdArgs)
				} else {
					for i, arg := range tt.wantArgs {
						if lastCmdArgs[i] != arg {
							t.Errorf("Expected arg[%d]=%s, got %s", i, arg, lastCmdArgs[i])
						}
					}
				}
			}
		})
	}
}

func TestGetLogFileDirectoryInternal_OS(t *testing.T) {
	isolateUserLog(t)

	t.Run("Windows With APPDATA", func(t *testing.T) {
		t.Setenv("APPDATA", "C:\\AppData")
		dir := getLogFileDirectoryInternal("windows")
		if dir != "C:\\AppData" {
			t.Errorf("Expected C:\\AppData, got %s", dir)
		}
	})

	t.Run("Windows Without APPDATA", func(t *testing.T) {
		t.Setenv("APPDATA", "")
		dir := getLogFileDirectoryInternal("windows")
		if dir != "." {
			t.Errorf("Expected ., got %s", dir)
		}
	})

	t.Run("Unix", func(t *testing.T) {
		t.Setenv("HOME", "/home/testuser")
		dir := getLogFileDirectoryInternal("linux")
		if dir != "/home/testuser" {
			t.Errorf("Expected /home/testuser, got %s", dir)
		}
	})
}

func TestSendWallMessageInternal_OS(t *testing.T) {
	setupTestLogger()

	oldExecCmdFunc := execCmdFunc
	defer func() { execCmdFunc = oldExecCmdFunc }()

	execCmdFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return harmlessTestCommand(ctx, "success")
	}

	t.Run("Unsupported OS", func(t *testing.T) {
		err := sendWallMessageInternal("test", "windows")
		if err == nil || !strings.Contains(err.Error(), "wall not supported on windows") {
			t.Errorf("Expected wall not supported error, got %v", err)
		}
	})

	t.Run("Supported OS", func(t *testing.T) {
		err := sendWallMessageInternal("test", "linux")
		if err != nil {
			// Might fail resolving wall on minimal environments
			if !strings.Contains(err.Error(), "executable not found") {
				t.Errorf("Unexpected error: %v", err)
			}
		}
	})
}

func TestProcessExistsInternal_OS(t *testing.T) {
	// Tests are difficult to make deterministic without mocking findProcessFunc,
	// which already exists in the file (var findProcessFunc).

	oldFindProcessFunc := findProcessFunc
	defer func() { findProcessFunc = oldFindProcessFunc }()

	t.Run("Unix Exists", func(t *testing.T) {
		// mock to succeed
		findProcessFunc = func(pid int) (processController, error) {
			return &mockProcess{signalErr: nil}, nil
		}
		if !processExistsInternal(1234, "linux") {
			t.Error("Expected process to exist")
		}
	})

	t.Run("Unix Not Exists", func(t *testing.T) {
		findProcessFunc = func(pid int) (processController, error) {
			return &mockProcess{signalErr: errors.New("not exists")}, nil
		}
		if processExistsInternal(1234, "linux") {
			t.Error("Expected process to not exist")
		}
	})

	t.Run("Windows Exists", func(t *testing.T) {
		findProcessFunc = func(pid int) (processController, error) {
			return &mockProcess{}, nil
		}
		if !processExistsInternal(1234, "windows") {
			t.Error("Expected process to exist")
		}
	})

	t.Run("Windows Not Exists", func(t *testing.T) {
		findProcessFunc = func(pid int) (processController, error) {
			return nil, errors.New("not exists")
		}
		if processExistsInternal(1234, "windows") {
			t.Error("Expected process to not exist")
		}
	})
}

// mockProcess implements processController for testing
type mockProcess struct {
	signalErr error
	killErr   error
}

func (m *mockProcess) Signal(sig os.Signal) error {
	return m.signalErr
}

func (m *mockProcess) Kill() error {
	return m.killErr
}
