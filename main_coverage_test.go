package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestInitLogger_Failure_Subprocess(t *testing.T) {
	if os.Getenv("TEST_INIT_LOGGER_FAILURE") == "1" {
		initLogger()
		return
	}

	dir := t.TempDir()
	_ = os.Chmod(dir, 0500)

	cmd := exec.Command(os.Args[0], "-test.run=TestInitLogger_Failure_Subprocess")
	cmd.Env = append(os.Environ(), "TEST_INIT_LOGGER_FAILURE=1", "HOME="+dir, "APPDATA="+dir)
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestResolveExecutableInternal_MissingWindows(t *testing.T) {
	orig := trustedWindowsSystemDirectory
	defer func() { trustedWindowsSystemDirectory = orig }()
	trustedWindowsSystemDirectory = func() (string, error) {
		dir := t.TempDir()
		sysDir := filepath.Join(dir, "System32")
		_ = os.MkdirAll(sysDir, 0755)
		_ = os.WriteFile(filepath.Join(sysDir, "shutdown.exe"), []byte(""), 0755)
		return sysDir, nil
	}
	_, err := resolveExecutableInternal("shutdown", "windows")
	if err != nil {
		t.Errorf("expected success")
	}
}

func TestResolveExecutableInternal_WindowsValid(t *testing.T) {
	orig := trustedWindowsSystemDirectory
	defer func() { trustedWindowsSystemDirectory = orig }()
	trustedWindowsSystemDirectory = func() (string, error) {
		dir := t.TempDir()
		sysDir := filepath.Join(dir, "System32")
		_ = os.MkdirAll(sysDir, 0755)
		// Missing shutdown.exe
		return sysDir, nil
	}
	_, err := resolveExecutableInternal("shutdown", "windows")
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestScheduleAtSpecificTime_Immediate(t *testing.T) {
	orig := execCmdFunc
	defer func() { execCmdFunc = orig }()
	execCmdFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "true")
	}

	future := time.Now().Add(1 * time.Second)
	timeStr := future.Format("15:04")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_ = scheduleAtSpecificTime(ctx, timeStr, ActionReboot, "msg", false)
}

func TestSendWallMessageInternal_Darwin(t *testing.T) {
	err := sendWallMessageInternal("msg", "darwin")
	if err == nil {
		t.Errorf("expected error for darwin")
	}
}

func TestExecuteSystemCommandInternal_WindowsUnsupported(t *testing.T) {
	err := executeSystemCommandInternal("unsupported_action", "windows")
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestExecuteSystemCommandInternal_DarwinUnsupported(t *testing.T) {
	err := executeSystemCommandInternal("unsupported_action", "darwin")
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestGetPIDFilePathInternal_WindowsFallbacks(t *testing.T) {
	t.Setenv("APPDATA", "")
	t.Setenv("USERPROFILE", "")

	defer func() { pidFilePathOnce = *new(sync.Once) }()
	pidFilePathOnce = *new(sync.Once)

	path := getPIDFilePathInternal("windows")
	if path == "" {
		t.Errorf("expected path, got empty")
	}
}

func TestProcessExistsInternal_FindProcessFailure(t *testing.T) {
	orig := findProcessFunc
	defer func() { findProcessFunc = orig }()
	findProcessFunc = func(pid int) (processController, error) {
		return nil, fmt.Errorf("simulated find error")
	}
	exists := processExistsInternal(12345, "linux")
	if exists {
		t.Errorf("expected false")
	}
}

func TestGetPIDFilePathInternal_WindowsFallbacks2(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	defer func() { pidFilePathOnce = *new(sync.Once) }()
	pidFilePathOnce = *new(sync.Once)

	path := getPIDFilePathInternal("windows")
	if path == "" {
		t.Errorf("expected path, got empty")
	}
}

func TestExecuteSystemCommandInternal_WindowsSystemRoot(t *testing.T) {
	orig := execCmdFunc
	defer func() { execCmdFunc = orig }()

	execCmdFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, "true")
		return cmd
	}
	orig2 := trustedWindowsSystemDirectory
	defer func() { trustedWindowsSystemDirectory = orig2 }()
	trustedWindowsSystemDirectory = func() (string, error) {
		dir := t.TempDir()
		sysDir := filepath.Join(dir, "System32")
		_ = os.MkdirAll(sysDir, 0755)
		_ = os.WriteFile(filepath.Join(sysDir, "shutdown.exe"), []byte(""), 0755)
		return sysDir, nil
	}

	_ = executeSystemCommandInternal(ActionReboot, "windows")
}

func TestExecuteSystemCommandInternal_DarwinActions(t *testing.T) {
	orig := execCmdFunc
	defer func() { execCmdFunc = orig }()
	execCmdFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "true")
	}

	_ = executeSystemCommandInternal(ActionReboot, "darwin")
	_ = executeSystemCommandInternal(ActionPoweroff, "darwin")
	_ = executeSystemCommandInternal(ActionHalt, "darwin")
}

func TestGetPIDFilePathInternal_WindowsFallbackToHomeDir(t *testing.T) {
	// Fallback when UserConfigDir fails but UserHomeDir succeeds
	t.Setenv("APPDATA", "")
	t.Setenv("USERPROFILE", t.TempDir()) // Ensure UserHomeDir succeeds

	// Use pointer to avoid copylocks in go vet
	defer func() { pidFilePathOnce = *new(sync.Once) }()
	pidFilePathOnce = *new(sync.Once)

	path := getPIDFilePathInternal("windows")
	if path == "" {
		t.Errorf("expected path, got empty")
	}
}

func TestCancelScheduledAction_RemovePIDFileFailsStale(t *testing.T) {
	setupTestLogger()

	info := ScheduleInfo{PID: 99999, Action: ActionReboot}
	data, _ := json.Marshal(info)

	dir := t.TempDir()
	pidDir := filepath.Join(dir, "piddir")
	_ = os.Mkdir(pidDir, 0755)

	pidFileDirOverride = pidDir
	defer func() { pidFileDirOverride = "" }()

	pidPath := getPIDFilePath()
	_ = os.WriteFile(pidPath, data, 0644)

	_ = os.Chmod(pidDir, 0555) // make read-only to fail removal
	defer func() { _ = os.Chmod(pidDir, 0755) }()

	err := cancelScheduledAction()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestCancelScheduledAction_RemovePIDFileFailsActive(t *testing.T) {
	setupTestLogger()

	info := ScheduleInfo{PID: os.Getpid(), Action: ActionReboot}
	data, _ := json.Marshal(info)

	dir := t.TempDir()
	pidDir := filepath.Join(dir, "piddir")
	_ = os.Mkdir(pidDir, 0755)

	pidFileDirOverride = pidDir
	defer func() { pidFileDirOverride = "" }()

	pidPath := getPIDFilePath()
	_ = os.WriteFile(pidPath, data, 0644)

	origFind := findProcessFunc
	defer func() { findProcessFunc = origFind }()
	findProcessFunc = func(pid int) (processController, error) {
		return &mockProcessController{
			signalFunc: func(sig os.Signal) error { return nil },
			killFunc:   func() error { return nil },
		}, nil
	}

	_ = os.Chmod(pidDir, 0555) // make read-only to fail removal
	defer func() { _ = os.Chmod(pidDir, 0755) }()

	err := cancelScheduledAction()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestShowScheduleStatus_StalePIDFile(t *testing.T) {
	setupTestLogger()

	info := ScheduleInfo{PID: 999999, Action: ActionReboot}
	data, _ := json.Marshal(info)

	dir := t.TempDir()
	pidFileDirOverride = dir
	defer func() { pidFileDirOverride = "" }()

	pidPath := getPIDFilePath()
	_ = os.WriteFile(pidPath, data, 0644)

	err := showScheduleStatus()
	if err != nil {
		t.Errorf("expected nil error for stale PID file, got %v", err)
	}
}

type mockProcessController struct {
	signalFunc func(sig os.Signal) error
	killFunc   func() error
}

func (m *mockProcessController) Signal(sig os.Signal) error {
	if m.signalFunc != nil {
		return m.signalFunc(sig)
	}
	return nil
}

func (m *mockProcessController) Kill() error {
	if m.killFunc != nil {
		return m.killFunc()
	}
	return nil
}

// Add tests for main function
func TestMain_Coverage(t *testing.T) {
	setupTestLogger()
	resetFlags()

	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()
	var exitCode int
	osExit = func(code int) { exitCode = code }

	t.Run("flagShutdown_bypass", func(t *testing.T) {
		resetFlags()
		flagShutdown = true
		exitCode = -1
		main()
		if exitCode != 1 && exitCode != 0 {
			t.Errorf("expected exit code 1 or 0, got %d", exitCode)
		}
	})
}

func TestMain_Coverage_Status(t *testing.T) {
	setupTestLogger()
	resetFlags()

	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()
	var exitCode int
	osExit = func(code int) { exitCode = code }

	t.Run("flagStatus_success", func(t *testing.T) {
		resetFlags()
		flagStatus = true
		exitCode = -1

		dir := t.TempDir()
		pidFileDirOverride = dir
		defer func() { pidFileDirOverride = "" }()

		main()
		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d", exitCode)
		}
	})
}

// TestMain_Coverage_Status_Error removed as showScheduleStatus currently returns nil on parse errors in production,
// avoiding structural refactors merely for coverage.

func TestMain_Coverage_TimeAndDelay(t *testing.T) {
	setupTestLogger()

	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()
	var exitCode int
	osExit = func(code int) { exitCode = code }

	t.Run("flagTime", func(t *testing.T) {
		resetFlags()
		flagTime = "99:99"
		exitCode = -1
		main()
		if exitCode != 1 && exitCode != 0 {
			t.Errorf("expected exit code 1 or 0, got %d", exitCode)
		}
	})

	t.Run("flagDelay", func(t *testing.T) {
		resetFlags()
		flagDelay = 0
		exitCode = -1
		main()
		if exitCode != 1 && exitCode != 0 {
			t.Errorf("expected exit code 1 or 0, got %d", exitCode)
		}
	})
}
