package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// WriteFakeExecGoCLI writes a fake execgocli that LookPath can resolve on all platforms.
// body should be a POSIX shell script (#!/bin/sh ...). On Windows it is wrapped by a .cmd
// that invokes Git Bash / bash, because CreateProcess cannot execute extensionless scripts.
func WriteFakeExecGoCLI(t testing.TB, body string) string {
	t.Helper()
	return WriteFakeShellCLI(t, "execgocli", body)
}

// WriteFakeShellCLI writes a named fake CLI (POSIX script, .cmd-wrapped on Windows).
func WriteFakeShellCLI(t testing.TB, name, body string) string {
	t.Helper()
	if name == "" {
		name = "fakecli"
	}
	dir := t.TempDir()
	script := filepath.Join(dir, name+".sh")
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		return script
	}
	bash, err := lookBash()
	if err != nil {
		t.Fatal(err)
	}
	cmdPath := filepath.Join(dir, name+".cmd")
	wrapper := fmt.Sprintf("@echo off\r\n\"%s\" \"%s\" %%*\r\n", bash, script)
	if err := os.WriteFile(cmdPath, []byte(wrapper), 0o755); err != nil {
		t.Fatal(err)
	}
	return cmdPath
}

func lookBash() (string, error) {
	if p, err := exec.LookPath("bash"); err == nil {
		return p, nil
	}
	for _, cand := range []string{
		`C:\Program Files\Git\bin\bash.exe`,
		`C:\Program Files\Git\usr\bin\bash.exe`,
	} {
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return cand, nil
		}
	}
	return "", fmt.Errorf("bash not found (required for fake CLI on Windows; install Git for Windows)")
}
