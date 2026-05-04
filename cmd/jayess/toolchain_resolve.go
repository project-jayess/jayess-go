package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"jayess-go/llvmbackend"
)

func resolveToolchainCommands(commands []llvmbackend.ToolchainCommand, targetName string) ([]llvmbackend.ToolchainCommand, error) {
	dirs := toolchainSearchDirs(targetName)
	resolved := append([]llvmbackend.ToolchainCommand{}, commands...)
	programs := map[string]string{}
	for index, command := range resolved {
		path, ok := programs[command.Program]
		if !ok {
			var err error
			path, err = resolveToolchainProgram(command.Program, dirs, targetName)
			if err != nil {
				return nil, err
			}
			programs[command.Program] = path
		}
		resolved[index].Program = path
	}
	return resolved, nil
}

func resolveToolchainProgram(program string, dirs []string, targetName string) (string, error) {
	if filepath.Base(program) != program {
		if fileIsExecutable(program) {
			return program, nil
		}
		return "", fmt.Errorf("missing toolchain tool %q", program)
	}
	for _, dir := range dirs {
		for _, name := range toolNames(program) {
			candidate := filepath.Join(dir, name)
			if fileIsExecutable(candidate) {
				return candidate, nil
			}
		}
	}
	if path, err := exec.LookPath(program); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("missing toolchain tool %q for %s; searched JAYESS_TOOLCHAIN, executable-local tools/%s, tools/%s, refs/llvm*/bin, refs/llvm-project*/bin, and PATH", program, targetName, targetName, targetName)
}

func toolNames(program string) []string {
	if filepath.Ext(program) == ".exe" {
		return []string{program}
	}
	return []string{program, program + ".exe"}
}

func fileIsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}
