//go:build !windows

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWindowsCommandExecutionOnOtherHosts(t *testing.T) {
	originalLogger := logger
	t.Cleanup(func() { logger = originalLogger })
	setupTestLogger()
	originalExec, originalDirectory := execCmdFunc, trustedWindowsSystemDirectory
	t.Cleanup(func() { execCmdFunc, trustedWindowsSystemDirectory = originalExec, originalDirectory })
	root := t.TempDir()
	directory := filepath.Join(root, "System32")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(directory, "shutdown.exe")
	if err := os.WriteFile(executable, nil, 0600); err != nil {
		t.Fatal(err)
	}
	trustedWindowsSystemDirectory = func() (string, error) { return directory, nil }
	for _, tc := range []struct {
		name, action, outcome string
		args                  []string
		wantErr               bool
	}{
		{"reboot", ActionReboot, "success", []string{"/r", "/t", "0"}, false},
		{"poweroff", ActionPoweroff, "success", []string{"/s", "/t", "0"}, false},
		{"halt", ActionHalt, "success", []string{"/s", "/t", "0"}, false},
		{"command failure", ActionHalt, "fail", []string{"/s", "/t", "0"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var command *exec.Cmd
			execCmdFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
				if name != executable || !reflect.DeepEqual(args, tc.args) {
					t.Errorf("unexpected command: %s %q", name, args)
				}
				command = harmlessTestCommand(ctx, tc.outcome)
				return command
			}
			err := executeSystemCommandInternal(tc.action, "windows")
			if (err != nil) != tc.wantErr {
				t.Fatalf("error=%v, wantErr=%v", err, tc.wantErr)
			}
			if command == nil {
				t.Fatal("command was not executed")
			}
			wantEnv := []string{"PATH=/usr/bin:/bin:/usr/sbin:/sbin", "SystemRoot=" + root}
			if !reflect.DeepEqual(command.Env, wantEnv) {
				t.Fatalf("unsafe environment: %q", command.Env)
			}
		})
	}
	t.Run("directory lookup failure prevents execution", func(t *testing.T) {
		expected := errors.New("directory unavailable")
		trustedWindowsSystemDirectory = func() (string, error) { return "", expected }
		execCmdFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
			t.Error("command attempted after lookup failed")
			return harmlessTestCommand(ctx, "fail")
		}
		if err := executeSystemCommandInternal(ActionHalt, "windows"); !errors.Is(err, expected) {
			t.Fatalf("expected lookup error, got %v", err)
		}
	})
}
