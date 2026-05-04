package main

import (
	"fmt"
	"os"
	"path/filepath"

	"jayess-go/llvmbackend"
	"jayess-go/tooling"
)

func outputPath(cfg cliConfig, target llvmbackend.TargetConfig) string {
	if cfg.outputPath != "" {
		return cfg.outputPath
	}
	base := moduleName(cfg.inputPath)
	name := tooling.DefaultOutputName(cfg.emit, target, base)
	switch cfg.emit {
	case tooling.EmitLLVMIR:
		name = base + ".ll"
	case tooling.EmitBitcode:
		name = base + ".bc"
	case tooling.EmitObject:
		name = base + ".o"
	case tooling.EmitStatic:
		name = "lib" + base + ".a"
	case tooling.EmitNative:
		if target.Name == "windows-x64" {
			name = base + ".exe"
		}
	case tooling.EmitDist:
		return filepath.Join("dist", base)
	}
	return filepath.Join("build", name)
}

func writeFile(path string, content []byte) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	return nil
}
