package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

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
	runtimePath, err := runtimeSourcePath("jayess_runtime.c")
	if err != nil {
		return fmt.Errorf("resolve runtime source: %w", err)
	}
	runtimeIncludeDir, err := runtimeIncludePath()
	if err != nil {
		return fmt.Errorf("resolve runtime include directory: %w", err)
	}
	brotliIncludeDir, brotliSources, brotliAvailable := brotliBuildInputs()

	if err := os.WriteFile(irPath, result.LLVMIR, 0o644); err != nil {
		return fmt.Errorf("write temporary LLVM IR: %w", err)
	}

	args := buildExecutableArgs(result, opts, irPath, runtimePath, runtimeIncludeDir, brotliIncludeDir, brotliSources, brotliAvailable, outputPath)
	clangCmd := exec.Command(tc.ClangPath, args...)
	if output, err := clangCmd.CombinedOutput(); err != nil {
		return formatNativeBuildError(err, string(output))
	}

	return nil
}

func buildExecutableArgs(result *compiler.Result, opts compiler.Options, irPath, runtimePath, runtimeIncludeDir, brotliIncludeDir string, brotliSources []string, brotliAvailable bool, outputPath string) []string {
	args := []string{"-target", opts.TargetTriple, "-I", runtimeIncludeDir}
	if optFlag := clangOptimizationFlag(opts.OptimizationLevel); optFlag != "" {
		args = append(args, optFlag)
	}
	args = append(args, clangTargetCodegenArgs(opts)...)
	for _, includeDir := range result.NativeIncludeDirs {
		args = append(args, "-I", includeDir)
	}
	args = append(args, result.NativeCompileFlags...)
	args = append(args, irPath, runtimePath)
	if brotliAvailable {
		args = append(args, "-I", brotliIncludeDir)
		args = append(args, brotliSources...)
	}
	args = append(args, result.NativeImports...)
	args = append(args, nativeSystemLinkFlags(opts.TargetTriple)...)
	args = append(args, result.NativeLinkFlags...)
	args = append(args, "-o", outputPath)
	return args
}

func nativeSystemLinkFlags(targetTriple string) []string {
	if strings.Contains(targetTriple, "windows") {
		return []string{"-lws2_32", "-lwinhttp", "-lsecur32", "-lcrypt32", "-lbcrypt"}
	}
	if strings.Contains(targetTriple, "linux") || strings.Contains(targetTriple, "darwin") {
		return []string{"-lssl", "-lcrypto", "-lz", "-lm"}
	}
	return nil
}

var undefinedSymbolPattern = regexp.MustCompile(`undefined reference to [` + "`" + `']([^` + "`" + `']+)[` + "`" + `']`)
var missingLibraryPattern = regexp.MustCompile(`cannot find -l([A-Za-z0-9_+.-]+)|library ['"]([^'"]+)['"] not found|unable to find library -l([A-Za-z0-9_+.-]+)`)
var missingHeaderPattern = regexp.MustCompile(`fatal error: ['"]([^'"]+\.(?:h|hh|hpp|hxx))['"] file not found|fatal error: ([^:\n]+?\.(?:h|hh|hpp|hxx)): No such file or directory`)

func formatNativeBuildError(err error, output string) error {
	if match := undefinedSymbolPattern.FindStringSubmatch(output); len(match) == 2 {
		return fmt.Errorf("native symbol resolution failed for %s: %w: %s", match[1], err, output)
	}
	if match := missingLibraryPattern.FindStringSubmatch(output); len(match) > 0 {
		for _, candidate := range match[1:] {
			if strings.TrimSpace(candidate) != "" {
				return fmt.Errorf("native library link failed for %s: %w: %s", candidate, err, output)
			}
		}
	}
	if strings.Contains(output, "Undefined symbols for architecture") {
		return fmt.Errorf("native symbol resolution failed: %w: %s", err, output)
	}
	if match := missingHeaderPattern.FindStringSubmatch(output); len(match) > 0 {
		for _, candidate := range match[1:] {
			if strings.TrimSpace(candidate) != "" {
				return fmt.Errorf("native header dependency missing for %s: %w: %s", candidate, err, output)
			}
		}
	}
	return fmt.Errorf("clang native build failed: %w: %s", err, output)
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
	runtimePath, err := runtimeSourcePath("jayess_runtime.c")
	if err != nil {
		return fmt.Errorf("resolve runtime source: %w", err)
	}
	runtimeIncludeDir, err := runtimeIncludePath()
	if err != nil {
		return fmt.Errorf("resolve runtime include directory: %w", err)
	}
	brotliIncludeDir, brotliSources, brotliAvailable := brotliBuildInputs()

	if err := os.WriteFile(irPath, result.LLVMIR, 0o644); err != nil {
		return fmt.Errorf("write temporary LLVM IR: %w", err)
	}

	args := buildSharedLibraryArgs(result, opts, irPath, runtimePath, runtimeIncludeDir, brotliIncludeDir, brotliSources, brotliAvailable, outputPath)
	clangCmd := exec.Command(tc.ClangPath, args...)
	if output, err := clangCmd.CombinedOutput(); err != nil {
		return formatNativeBuildError(err, string(output))
	}

	return nil
}

func buildSharedLibraryArgs(result *compiler.Result, opts compiler.Options, irPath, runtimePath, runtimeIncludeDir, brotliIncludeDir string, brotliSources []string, brotliAvailable bool, outputPath string) []string {
	args := []string{"-target", opts.TargetTriple, "-I", runtimeIncludeDir}
	if optFlag := clangOptimizationFlag(opts.OptimizationLevel); optFlag != "" {
		args = append(args, optFlag)
	}
	args = append(args, clangTargetCodegenArgs(opts)...)
	args = append(args, sharedLibraryModeArgs(opts.TargetTriple)...)
	for _, includeDir := range result.NativeIncludeDirs {
		args = append(args, "-I", includeDir)
	}
	args = append(args, result.NativeCompileFlags...)
	args = append(args, irPath, runtimePath)
	if brotliAvailable {
		args = append(args, "-I", brotliIncludeDir)
		args = append(args, brotliSources...)
	}
	args = append(args, result.NativeImports...)
	args = append(args, nativeSystemLinkFlags(opts.TargetTriple)...)
	args = append(args, result.NativeLinkFlags...)
	args = append(args, "-o", outputPath)
	return args
}

func sharedLibraryModeArgs(targetTriple string) []string {
	if strings.Contains(targetTriple, "darwin") {
		return []string{"-dynamiclib", "-fPIC"}
	}
	return []string{"-shared", "-fPIC"}
}

func clangOptimizationFlag(level string) string {
	switch level {
	case "", "O0":
		return "-O0"
	case "O1":
		return "-O1"
	case "O2":
		return "-O2"
	case "O3":
		return "-O3"
	case "Oz":
		return "-Oz"
	default:
		return ""
	}
}

func clangTargetCodegenArgs(opts compiler.Options) []string {
	var args []string
	if opts.TargetCPU != "" {
		args = append(args, "-mcpu="+opts.TargetCPU)
	}
	for _, feature := range opts.TargetFeatures {
		feature = strings.TrimSpace(feature)
		if feature == "" {
			continue
		}
		args = append(args, "-Xclang", "-target-feature", "-Xclang", feature)
	}
	switch opts.RelocationModel {
	case "pic":
		args = append(args, "-fPIC")
	case "pie":
		args = append(args, "-fPIE")
	case "static":
		args = append(args, "-fno-pic")
	}
	if opts.CodeModel != "" {
		args = append(args, "-mcmodel="+opts.CodeModel)
	}
	return args
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

func brotliBuildInputs() (string, []string, bool) {
	runtimeDir, err := runtimeIncludePath()
	if err != nil {
		return "", nil, false
	}
	rootDir := filepath.Dir(runtimeDir)
	brotliRoot := filepath.Join(rootDir, "refs", "brotli")
	includeDir := filepath.Join(brotliRoot, "c", "include")
	if _, err := os.Stat(includeDir); err != nil {
		return "", nil, false
	}
	patterns := []string{
		filepath.Join(brotliRoot, "c", "common", "*.c"),
		filepath.Join(brotliRoot, "c", "dec", "*.c"),
		filepath.Join(brotliRoot, "c", "enc", "*.c"),
	}
	var sources []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil || len(matches) == 0 {
			return "", nil, false
		}
		sources = append(sources, matches...)
	}
	return includeDir, sources, true
}
