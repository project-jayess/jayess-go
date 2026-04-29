package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"jayess-go/compiler"
)

type Toolchain struct {
	ClangPath  string
	LLVMAsPath string
	ARPath     string
}

func DetectToolchain() (*Toolchain, error) {
	clangPath, err := exec.LookPath("clang")
	if err != nil {
		return nil, fmt.Errorf("clang was not found in PATH")
	}

	return &Toolchain{
		ClangPath:  clangPath,
		LLVMAsPath: lookupOptionalTool("llvm-as"),
		ARPath:     lookupArchiveTool(),
	}, nil
}

func lookupOptionalTool(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

func lookupArchiveTool() string {
	for _, name := range []string{"llvm-ar", "ar"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path
		}
	}
	return ""
}

func (tc *Toolchain) BuildExecutable(result *compiler.Result, opts compiler.Options, outputPath string) error {
	tempDir, err := os.MkdirTemp("", "jayess-llvm-*")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	irPath := filepath.Join(tempDir, "module.ll")
	runtimePaths, err := runtimeSourcePaths()
	if err != nil {
		return fmt.Errorf("resolve runtime sources: %w", err)
	}
	runtimeIncludeDir, err := runtimeIncludePath()
	if err != nil {
		return fmt.Errorf("resolve runtime include directory: %w", err)
	}
	brotliIncludeDir, brotliSources, brotliAvailable := brotliBuildInputs()

	if err := os.WriteFile(irPath, result.LLVMIR, 0o644); err != nil {
		return fmt.Errorf("write temporary LLVM IR: %w", err)
	}

	objectPaths, err := tc.buildNativeObjectSet(tempDir, result, opts, irPath, runtimePaths, runtimeIncludeDir, brotliIncludeDir, brotliSources, brotliAvailable)
	if err != nil {
		return err
	}

	args := []string{"-target", opts.TargetTriple}
	args = append(args, objectPaths...)
	args = append(args, nativeSystemLinkFlags(opts.TargetTriple)...)
	args = append(args, result.NativeLinkFlags...)
	args = append(args, "-o", outputPath)
	clangCmd := exec.Command(tc.ClangPath, args...)
	if output, err := clangCmd.CombinedOutput(); err != nil {
		return formatNativeBuildErrorForTarget(err, string(output), opts.TargetTriple)
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

	args := []string{"-target", opts.TargetTriple}
	if llvmIRContainsDebugMetadata(result.LLVMIR) {
		args = append(args, "-g")
	}
	if optFlag := clangOptimizationFlag(opts.OptimizationLevel); optFlag != "" {
		args = append(args, optFlag)
	}
	args = append(args, clangTargetCodegenArgs(opts)...)
	args = append(args, "-c", irPath, "-o", outputPath)
	clangCmd := exec.Command(tc.ClangPath, args...)
	if output, err := clangCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("clang object build failed: %w: %s", err, string(output))
	}

	return nil
}

func (tc *Toolchain) BuildBitcode(result *compiler.Result, outputPath string) error {
	if tc.LLVMAsPath == "" {
		return fmt.Errorf("llvm-as was not found in PATH")
	}

	tempDir, err := os.MkdirTemp("", "jayess-llvm-*")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	irPath := filepath.Join(tempDir, "module.ll")
	if err := os.WriteFile(irPath, result.LLVMIR, 0o644); err != nil {
		return fmt.Errorf("write temporary LLVM IR: %w", err)
	}

	cmd := exec.Command(tc.LLVMAsPath, irPath, "-o", outputPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("llvm-as bitcode build failed: %w: %s", err, string(output))
	}
	return nil
}

func (tc *Toolchain) BuildStaticLibrary(result *compiler.Result, opts compiler.Options, outputPath string) error {
	if tc.ARPath == "" {
		return fmt.Errorf("ar or llvm-ar was not found in PATH")
	}

	tempDir, err := os.MkdirTemp("", "jayess-llvm-*")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	objectPath := filepath.Join(tempDir, "module.o")
	if err := tc.BuildObject(result, opts, objectPath); err != nil {
		return err
	}

	cmd := exec.Command(tc.ARPath, "rcs", outputPath, objectPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("archive static library failed: %w: %s", err, string(output))
	}
	return nil
}

func (tc *Toolchain) BuildSharedLibrary(result *compiler.Result, opts compiler.Options, outputPath string) error {
	tempDir, err := os.MkdirTemp("", "jayess-llvm-*")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	irPath := filepath.Join(tempDir, "module.ll")
	runtimePaths, err := runtimeSourcePaths()
	if err != nil {
		return fmt.Errorf("resolve runtime sources: %w", err)
	}
	runtimeIncludeDir, err := runtimeIncludePath()
	if err != nil {
		return fmt.Errorf("resolve runtime include directory: %w", err)
	}
	brotliIncludeDir, brotliSources, brotliAvailable := brotliBuildInputs()

	if err := os.WriteFile(irPath, result.LLVMIR, 0o644); err != nil {
		return fmt.Errorf("write temporary LLVM IR: %w", err)
	}

	sharedOpts := opts
	if sharedOpts.RelocationModel == "" {
		sharedOpts.RelocationModel = "pic"
	}
	objectPaths, err := tc.buildNativeObjectSet(tempDir, result, sharedOpts, irPath, runtimePaths, runtimeIncludeDir, brotliIncludeDir, brotliSources, brotliAvailable)
	if err != nil {
		return err
	}

	args := []string{"-target", sharedOpts.TargetTriple}
	args = append(args, sharedLibraryModeArgs(sharedOpts.TargetTriple)...)
	args = append(args, objectPaths...)
	args = append(args, nativeSystemLinkFlags(sharedOpts.TargetTriple)...)
	args = append(args, result.NativeLinkFlags...)
	args = append(args, "-o", outputPath)
	clangCmd := exec.Command(tc.ClangPath, args...)
	if output, err := clangCmd.CombinedOutput(); err != nil {
		return formatNativeBuildErrorForTarget(err, string(output), sharedOpts.TargetTriple)
	}

	return nil
}
