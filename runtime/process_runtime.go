package runtime

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"time"
)

type ProcessRuntime struct {
	args     []string
	env      map[string]string
	stdin    *IOStream
	stdout   *IOStream
	stderr   *IOStream
	exitCode int
	exited   bool
	start    time.Time
}

func NewProcessRuntime(args []string, env map[string]string, stdin *IOStream, stdout *IOStream, stderr *IOStream) *ProcessRuntime {
	if env == nil {
		env = osEnvironment()
	}
	return &ProcessRuntime{
		args:   append([]string{}, args...),
		env:    copyStringMap(env),
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		start:  time.Now(),
	}
}

func NewProcessRuntimeFromIO(args []string, env map[string]string, stdin io.Reader, stdout io.Writer, stderr io.Writer) *ProcessRuntime {
	return NewProcessRuntime(
		args,
		env,
		NewReadableStream("stdin", stdin),
		NewWritableStream("stdout", stdout),
		NewWritableStream("stderr", stderr),
	)
}

func NewDefaultProcessRuntime() *ProcessRuntime {
	return NewProcessRuntimeFromIO(os.Args, nil, os.Stdin, os.Stdout, os.Stderr)
}

func NewBufferedProcessRuntime(args []string, env map[string]string, input string) (*ProcessRuntime, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return NewProcessRuntime(
		args,
		env,
		NewReadableStream("stdin", bytes.NewBufferString(input)),
		NewWritableStream("stdout", stdout),
		NewWritableStream("stderr", stderr),
	), stdout, stderr
}

func (process *ProcessRuntime) Argv() []string {
	return append([]string{}, process.args...)
}

func (process *ProcessRuntime) Env() map[string]string {
	return copyStringMap(process.env)
}

func (process *ProcessRuntime) Cwd() (string, error) {
	return os.Getwd()
}

func (process *ProcessRuntime) Exit(code int) {
	process.exitCode = code
	process.exited = true
}

func (process *ProcessRuntime) ExitCode() int {
	return process.exitCode
}

func (process *ProcessRuntime) Exited() bool {
	return process.exited
}

func (process *ProcessRuntime) Stdin() *IOStream {
	return process.stdin
}

func (process *ProcessRuntime) Stdout() *IOStream {
	return process.stdout
}

func (process *ProcessRuntime) Stderr() *IOStream {
	return process.stderr
}

func (process *ProcessRuntime) PID() int {
	return os.Getpid()
}

func (process *ProcessRuntime) Platform() string {
	return runtime.GOOS
}

func (process *ProcessRuntime) HRTime() time.Duration {
	return time.Since(process.start)
}

func osEnvironment() map[string]string {
	values := map[string]string{}
	for _, entry := range os.Environ() {
		for index, ch := range entry {
			if ch == '=' {
				values[entry[:index]] = entry[index+1:]
				break
			}
		}
	}
	return values
}

func copyStringMap(values map[string]string) map[string]string {
	copied := map[string]string{}
	for key, value := range values {
		copied[key] = value
	}
	return copied
}
