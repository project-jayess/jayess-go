package main

import (
	"fmt"
	"os"

	"jayess-go/llvmbackend"
	"jayess-go/llvmc"
)

const objectFileWorkDir = "temp/jayess-build"

func compileObjectFile(ir string, target llvmbackend.TargetConfig, outputPath string) error {
	if outputPath == "" {
		return fmt.Errorf("missing object output path")
	}
	if err := ensureParentDir(outputPath); err != nil {
		return err
	}
	if llvmc.Available() {
		return llvmc.EmitObject(llvmc.ObjectRequest{
			IR:           ir,
			TargetTriple: target.Triple,
			OutputPath:   outputPath,
		})
	}
	command := llvmbackend.ObjectFileIRCommand(target, llvmbackend.ObjectFileIRPath(outputPath, objectFileWorkDir), outputPath)
	commands, err := resolveToolchainCommands([]llvmbackend.ToolchainCommand{command}, target.Name)
	if err != nil {
		return err
	}
	irPath := llvmbackend.ObjectFileIRPath(outputPath, objectFileWorkDir)
	if err := writeFile(irPath, []byte(ir)); err != nil {
		return err
	}
	for _, command := range commands {
		if err := runToolchainCommand(command); err != nil {
			return err
		}
	}
	if _, err := os.Stat(outputPath); err != nil {
		return fmt.Errorf("object output was not produced: %w", err)
	}
	return nil
}
