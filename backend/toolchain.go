package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"jayess-go/compiler"
)

type Toolchain struct {
	ClangPath string
}

func DetectToolchain() (*Toolchain, error) {
	clangPath, err := exec.LookPath("clang")
	if err != nil {
		return nil, fmt.Errorf("clang was not found in PATH")
	}

	return &Toolchain{
		ClangPath: clangPath,
	}, nil
}

func (tc *Toolchain) BuildExecutable(inputPath string, opts compiler.Options, outputPath string) error {
	result, err := compiler.CompilePath(inputPath, opts)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "jayess-llvm-*")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	irPath := filepath.Join(tempDir, "module.ll")
	runtimePath, err := filepath.Abs(filepath.Join("runtime", "jayess_runtime.c"))
	if err != nil {
		return fmt.Errorf("resolve runtime source: %w", err)
	}
	runtimeIncludeDir, err := filepath.Abs("runtime")
	if err != nil {
		return fmt.Errorf("resolve runtime include directory: %w", err)
	}

	if err := os.WriteFile(irPath, result.LLVMIR, 0o644); err != nil {
		return fmt.Errorf("write temporary LLVM IR: %w", err)
	}

	args := []string{"-target", opts.TargetTriple, "-I", runtimeIncludeDir, irPath, runtimePath}
	args = append(args, result.NativeImports...)
	args = append(args, "-o", outputPath)
	clangCmd := exec.Command(tc.ClangPath, args...)
	if output, err := clangCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("clang native build failed: %w: %s", err, string(output))
	}

	return nil
}
