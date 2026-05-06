package runtime

import (
	"bytes"
	"os"
	"os/exec"
)

type ChildProcess struct {
	command *exec.Cmd
	stdout  bytes.Buffer
	stderr  bytes.Buffer
}

type ChildResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func SpawnProcess(command string, args []string, env map[string]string, stdin string) (*ChildProcess, error) {
	cmd := exec.Command(command, args...)
	child := &ChildProcess{command: cmd}
	cmd.Stdout = &child.stdout
	cmd.Stderr = &child.stderr
	if stdin != "" {
		cmd.Stdin = bytes.NewBufferString(stdin)
	}
	if env != nil {
		cmd.Env = environmentList(env)
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return child, nil
}

func ExecProcess(command string, args []string, env map[string]string, stdin string) (ChildResult, error) {
	child, err := SpawnProcess(command, args, env, stdin)
	if err != nil {
		return ChildResult{}, err
	}
	err = child.Wait()
	return ChildResult{
		Stdout:   child.Stdout(),
		Stderr:   child.Stderr(),
		ExitCode: child.ExitStatus(),
	}, err
}

func (child *ChildProcess) Wait() error {
	if child == nil || child.command == nil {
		return nil
	}
	return child.command.Wait()
}

func (child *ChildProcess) Stdout() string {
	if child == nil {
		return ""
	}
	return child.stdout.String()
}

func (child *ChildProcess) Stderr() string {
	if child == nil {
		return ""
	}
	return child.stderr.String()
}

func (child *ChildProcess) ExitStatus() int {
	if child == nil || child.command == nil || child.command.ProcessState == nil {
		return -1
	}
	return child.command.ProcessState.ExitCode()
}

func (child *ChildProcess) Signal(signal os.Signal) error {
	if child == nil || child.command == nil || child.command.Process == nil {
		return nil
	}
	return child.command.Process.Signal(signal)
}

func (child *ChildProcess) Cleanup() error {
	if child == nil || child.command == nil || child.command.Process == nil {
		return nil
	}
	if child.command.ProcessState != nil && child.command.ProcessState.Exited() {
		return nil
	}
	return child.command.Process.Kill()
}

func environmentList(values map[string]string) []string {
	env := make([]string, 0, len(values))
	for key, value := range values {
		env = append(env, key+"="+value)
	}
	return env
}
