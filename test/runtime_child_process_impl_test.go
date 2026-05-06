package test

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestChildProcessExecCapturesStreamsAndExitCode(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable returned error: %v", err)
	}
	result, err := jayessruntime.ExecProcess(
		executable,
		[]string{"-test.run=TestChildProcessRuntimeHelper", "--", "exec", "7"},
		childProcessTestEnv(),
		"",
	)
	if err == nil {
		t.Fatalf("expected non-zero helper exit to return an error")
	}
	if result.Stdout != "exec stdout\n" || result.Stderr != "exec stderr\n" || result.ExitCode != 7 {
		t.Fatalf("unexpected child result: %#v", result)
	}
}

func TestChildProcessSpawnFeedsStdinAndWaits(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable returned error: %v", err)
	}
	child, err := jayessruntime.SpawnProcess(
		executable,
		[]string{"-test.run=TestChildProcessRuntimeHelper", "--", "stdin", "0"},
		childProcessTestEnv(),
		"from stdin",
	)
	if err != nil {
		t.Fatalf("SpawnProcess returned error: %v", err)
	}
	if err := child.Wait(); err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}
	if child.ExitStatus() != 0 || child.Stdout() != "from stdin" {
		t.Fatalf("unexpected child state exit=%d stdout=%q stderr=%q", child.ExitStatus(), child.Stdout(), child.Stderr())
	}
}

func TestChildProcessRuntimeHelper(t *testing.T) {
	if os.Getenv("JAYESS_CHILD_PROCESS_HELPER") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	if len(args) < 3 {
		os.Exit(2)
	}
	mode := args[1]
	code, err := strconv.Atoi(args[2])
	if err != nil {
		os.Exit(2)
	}
	switch mode {
	case "exec":
		fmt.Fprintln(os.Stdout, "exec stdout")
		fmt.Fprintln(os.Stderr, "exec stderr")
	case "stdin":
		_, _ = io.Copy(os.Stdout, os.Stdin)
	}
	os.Exit(code)
}

func childProcessTestEnv() map[string]string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		for index, ch := range entry {
			if ch == '=' {
				env[entry[:index]] = entry[index+1:]
				break
			}
		}
	}
	env["JAYESS_CHILD_PROCESS_HELPER"] = "1"
	return env
}
