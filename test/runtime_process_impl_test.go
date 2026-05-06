package test

import (
	"bytes"
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestProcessRuntimeStdioExitAndInfo(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	process := jayessruntime.NewProcessRuntimeFromIO(
		[]string{"jayess", "build"},
		map[string]string{"JAYESS_ENV": "test"},
		bytes.NewBufferString("input"),
		&stdout,
		&stderr,
	)

	input, err := process.Stdin().ReadAll()
	if err != nil {
		t.Fatalf("stdin ReadAll returned error: %v", err)
	}
	if string(input) != "input" {
		t.Fatalf("unexpected stdin content %q", string(input))
	}
	if _, err := process.Stdout().WriteString("out"); err != nil {
		t.Fatalf("stdout WriteString returned error: %v", err)
	}
	if _, err := process.Stderr().WriteString("err"); err != nil {
		t.Fatalf("stderr WriteString returned error: %v", err)
	}
	if stdout.String() != "out" || stderr.String() != "err" {
		t.Fatalf("unexpected stdio stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	process.Exit(42)
	if !process.Exited() || process.ExitCode() != 42 {
		t.Fatalf("unexpected exit state exited=%v code=%d", process.Exited(), process.ExitCode())
	}
	if process.PID() <= 0 || process.Platform() == "" || process.HRTime() < 0 {
		t.Fatalf("process info was not populated")
	}
	if _, err := process.Cwd(); err != nil {
		t.Fatalf("Cwd returned error: %v", err)
	}
	if process.Env()["JAYESS_ENV"] != "test" || process.Argv()[1] != "build" {
		t.Fatalf("unexpected process argv/env")
	}
}
