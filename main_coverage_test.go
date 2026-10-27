package main

import (
	"context"
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

	defer func() { pidFilePathOnce = sync.Once{} }()
	pidFilePathOnce = sync.Once{}

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

	defer func() { pidFilePathOnce = sync.Once{} }()
	pidFilePathOnce = sync.Once{}

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
