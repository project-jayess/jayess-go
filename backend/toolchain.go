package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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

func (tc *Toolchain) BuildExecutable(result *compiler.Result, opts compiler.Options, outputPath string) error {
	tempDir, err := os.MkdirTemp("", "jayess-llvm-*")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	irPath := filepath.Join(tempDir, "module.ll")
	runtimePath, err := runtimeSourcePath("jayess_runtime.c")
	if err != nil {
		return fmt.Errorf("resolve runtime source: %w", err)
	}
	runtimeIncludeDir, err := runtimeIncludePath()
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

func (tc *Toolchain) BuildObject(result *compiler.Result, opts compiler.Options, outputPath string) error {
	tempDir, err := os.MkdirTemp("", "jayess-llvm-*")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	irPath := filepath.Join(tempDir, "module.ll")
	if err := os.WriteFile(irPath, result.LLVMIR, 0o644); err != nil {
		return fmt.Errorf("write temporary LLVM IR: %w", err)
	}

	args := []string{"-target", opts.TargetTriple, "-c", irPath, "-o", outputPath}
	clangCmd := exec.Command(tc.ClangPath, args...)
	if output, err := clangCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("clang object build failed: %w: %s", err, string(output))
	}

	return nil
}

func runtimeSourcePath(name string) (string, error) {
	base, err := runtimeIncludePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, name), nil
}

func runtimeIncludePath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve backend source location")
	}
	return filepath.Join(filepath.Dir(filepath.Dir(file)), "runtime"), nil
}
