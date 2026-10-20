package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakefileFormattingAndToolchain(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make is not installed")
	}
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, name := range []string{"Makefile", ".go-version"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "example.go"), []byte("package example\nfunc Example( ){ }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(wantSuccess bool, args ...string) string {
		t.Helper()
		cmd := exec.Command("make", append([]string{"--no-print-directory", "GO=" + goBinary}, args...)...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if (err == nil) != wantSuccess {
			t.Fatalf("make %v: %v\n%s", args, err, output)
		}
		return string(output)
	}
	if output := run(false, "format-check"); !strings.Contains(output, "example.go") {
		t.Fatalf("format-check did not report the unformatted source: %s", output)
	}
	run(true, "format")
	run(true, "format-check")
	version, err := os.ReadFile(".go-version")
	if err != nil {
		t.Fatal(err)
	}
	output := run(false, "doctor", "GO_VERSION=0.0.0", "GO_TOOLCHAIN=go"+strings.TrimSpace(string(version)))
	if !strings.Contains(output, "Go toolchain mismatch") {
		t.Fatalf("doctor did not reject the mismatched toolchain: %s", output)
	}
}
