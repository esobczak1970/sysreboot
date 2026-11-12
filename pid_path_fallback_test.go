package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Recover PID-location scenarios from the retired fallback-test branch
// without resetting sync.Once state in the main test process.
func TestPIDFilePathFallbackIsolated(t *testing.T) {
	for _, scenario := range []string{"user-config", "primary"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			cmd := exec.Command(os.Args[0], "-test.run=^TestPIDFilePathFallbackChild$")
			cmd.Env = append(os.Environ(),
				"SYSREBOOT_PID_PATH_TEST_SCENARIO="+scenario,
				"SYSREBOOT_PID_PATH_TEST_ROOT="+root,
				"APPDATA="+root,
				"XDG_CONFIG_HOME="+root,
				"HOME="+root,
			)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("PID path subprocess failed: %v\n%s", err, output)
			}
		})
	}
}

func TestPIDFilePathFallbackChild(t *testing.T) {
	scenario := os.Getenv("SYSREBOOT_PID_PATH_TEST_SCENARIO")
	if scenario == "" {
		return
	}
	root := os.Getenv("SYSREBOOT_PID_PATH_TEST_ROOT")
	if root == "" {
		t.Fatal("missing isolated PID test root")
	}

	// Every child starts with fresh sync.Once and cached-path state.
	pidFileDirOverride = ""
	if scenario == "primary" {
		want := filepath.Join(pidFileDir, pidFileName)
		if got := getPIDFilePathInternal("linux"); got != want {
			t.Fatalf("non-Windows PID path = %q; want %q", got, want)
		}
		return
	}
	if scenario != "user-config" {
		t.Fatalf("unknown PID path scenario %q", scenario)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("isolated user config directory unavailable: %v", err)
	}
	if !strings.HasPrefix(configDir, root+string(os.PathSeparator)) && configDir != root {
		t.Fatalf("user config directory %q escaped test root %q", configDir, root)
	}

	want := filepath.Join(configDir, appName, pidFileName)
	if got := getPIDFilePathInternal("windows"); got != want {
		t.Fatalf("Windows-style PID path = %q; want %q", got, want)
	}
	if info, err := os.Stat(filepath.Dir(want)); err != nil || !info.IsDir() {
		t.Fatalf("PID directory not created: %v", err)
	}
	if got := getPIDFilePathInternal("linux"); got != want {
		t.Fatalf("cached PID path changed from %q to %q", want, got)
	}
}

func TestPIDFilePathOverrideBypassesCache(t *testing.T) {
	old := pidFileDirOverride
	pidFileDirOverride = t.TempDir()
	t.Cleanup(func() { pidFileDirOverride = old })

	want := filepath.Join(pidFileDirOverride, pidFileName)
	for _, goos := range []string{"windows", "linux", "darwin"} {
		if got := getPIDFilePathInternal(goos); got != want {
			t.Errorf("override with GOOS %q = %q; want %q", goos, got, want)
		}
	}
}
