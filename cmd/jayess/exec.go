package main

import (
	"fmt"
	"os/exec"
	"strings"

	"jayess-go/llvmbackend"
)

func runToolchainCommand(command llvmbackend.ToolchainCommand) error {
	process := exec.Command(command.Program, command.Args...)
	output, err := process.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			return fmt.Errorf("%s failed: %w", command.Step, err)
		}
		return fmt.Errorf("%s failed: %w: %s", command.Step, err, detail)
	}
	return nil
}
