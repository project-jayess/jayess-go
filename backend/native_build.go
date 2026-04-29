package backend

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"jayess-go/compiler"
)

// Native build plumbing for LLVM IR plus runtime/native source compilation.

type cachedSupportObjectSet struct {
	ready       chan struct{}
	objectPaths []string
	err         error
}

var sharedSupportObjectCache = struct {
	mu      sync.Mutex
	entries map[string]*cachedSupportObjectSet
}{
	entries: map[string]*cachedSupportObjectSet{},
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

func (tc *Toolchain) buildNativeObjectSet(tempDir string, result *compiler.Result, opts compiler.Options, irPath string, runtimePaths []string, runtimeIncludeDir, brotliIncludeDir string, brotliSources []string, brotliAvailable bool) ([]string, error) {
	var objectPaths []string
	debugEnabled := llvmIRContainsDebugMetadata(result.LLVMIR)

	moduleObjectPath := filepath.Join(tempDir, "module.o")
	if err := tc.compileIRToObject(irPath, opts, moduleObjectPath, debugEnabled); err != nil {
		return nil, err
	}
	objectPaths = append(objectPaths, moduleObjectPath)

	supportObjectPaths, err := tc.cachedSupportObjectPaths(opts, runtimePaths, runtimeIncludeDir, brotliIncludeDir, brotliSources, brotliAvailable, debugEnabled)
	if err != nil {
		return nil, err
	}
	objectPaths = append(objectPaths, supportObjectPaths...)

	nativeIncludeDirs := append([]string{runtimeIncludeDir}, result.NativeIncludeDirs...)
	for i, source := range result.NativeImports {
		objectPath := filepath.Join(tempDir, fmt.Sprintf("native-%d.o", i))
		if err := tc.compileNativeSourceToObject(source, opts, nativeIncludeDirs, result.NativeCompileFlags, objectPath, debugEnabled); err != nil {
			return nil, err
		}
		objectPaths = append(objectPaths, objectPath)
	}

	return objectPaths, nil
}

func (tc *Toolchain) cachedSupportObjectPaths(opts compiler.Options, runtimePaths []string, runtimeIncludeDir, brotliIncludeDir string, brotliSources []string, brotliAvailable bool, debugEnabled bool) ([]string, error) {
	keyParts := []string{
		tc.ClangPath,
		opts.TargetTriple,
		opts.OptimizationLevel,
		opts.RelocationModel,
		opts.CodeModel,
		fmt.Sprintf("debug=%t", debugEnabled),
		fmt.Sprintf("brotli=%t", brotliAvailable),
	}
	keyParts = append(keyParts, clangTargetCodegenArgs(opts)...)
	keyParts = append(keyParts, runtimePaths...)
	if brotliAvailable {
		keyParts = append(keyParts, brotliIncludeDir)
		keyParts = append(keyParts, brotliSources...)
	}
	cacheKey := strings.Join(keyParts, "\x00")

	sharedSupportObjectCache.mu.Lock()
	entry, ok := sharedSupportObjectCache.entries[cacheKey]
	if !ok {
		entry = &cachedSupportObjectSet{ready: make(chan struct{})}
		sharedSupportObjectCache.entries[cacheKey] = entry
		sharedSupportObjectCache.mu.Unlock()

		entry.objectPaths, entry.err = tc.compileSupportObjectSet(opts, runtimePaths, runtimeIncludeDir, brotliIncludeDir, brotliSources, brotliAvailable, debugEnabled)
		close(entry.ready)
	} else {
		sharedSupportObjectCache.mu.Unlock()
		<-entry.ready
	}

	if entry.err != nil {
		return nil, entry.err
	}
	return append([]string(nil), entry.objectPaths...), nil
}

func (tc *Toolchain) compileSupportObjectSet(opts compiler.Options, runtimePaths []string, runtimeIncludeDir, brotliIncludeDir string, brotliSources []string, brotliAvailable bool, debugEnabled bool) ([]string, error) {
	cacheDir, err := os.MkdirTemp("", "jayess-runtime-cache-*")
	if err != nil {
		return nil, fmt.Errorf("create support object cache directory: %w", err)
	}

	var objectPaths []string
	runtimeIncludeDirs := []string{runtimeIncludeDir}
	if brotliAvailable {
		runtimeIncludeDirs = append(runtimeIncludeDirs, brotliIncludeDir)
	}
	for i, runtimePath := range runtimePaths {
		runtimeObjectPath := filepath.Join(cacheDir, fmt.Sprintf("runtime-%d.o", i))
		if err := tc.compileNativeSourceToObject(runtimePath, opts, runtimeIncludeDirs, nil, runtimeObjectPath, debugEnabled); err != nil {
			return nil, err
		}
		objectPaths = append(objectPaths, runtimeObjectPath)
	}

	if brotliAvailable {
		for i, source := range brotliSources {
			objectPath := filepath.Join(cacheDir, fmt.Sprintf("brotli-%d.o", i))
			if err := tc.compileNativeSourceToObject(source, opts, []string{brotliIncludeDir}, nil, objectPath, debugEnabled); err != nil {
				return nil, err
			}
			objectPaths = append(objectPaths, objectPath)
		}
	}

	return objectPaths, nil
}

func (tc *Toolchain) compileIRToObject(irPath string, opts compiler.Options, outputPath string, debugEnabled bool) error {
	args := []string{"-target", opts.TargetTriple}
	if debugEnabled {
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

func (tc *Toolchain) compileNativeSourceToObject(sourcePath string, opts compiler.Options, includeDirs, compileFlags []string, outputPath string, debugEnabled bool) error {
	args := []string{"-target", opts.TargetTriple}
	if debugEnabled {
		args = append(args, "-g")
	}
	if optFlag := clangOptimizationFlag(opts.OptimizationLevel); optFlag != "" {
		args = append(args, optFlag)
	}
	args = append(args, clangTargetCodegenArgs(opts)...)
	for _, includeDir := range includeDirs {
		args = append(args, "-I", includeDir)
	}
	args = append(args, compileFlags...)
	args = append(args, "-c", sourcePath, "-o", outputPath)
	clangCmd := exec.Command(tc.ClangPath, args...)
	if output, err := clangCmd.CombinedOutput(); err != nil {
		return formatNativeBuildErrorForTarget(err, string(output), opts.TargetTriple)
	}
	return nil
}

func llvmIRContainsDebugMetadata(llvmIR []byte) bool {
	return bytes.Contains(llvmIR, []byte("!llvm.dbg.cu"))
}

func runtimeSourcePath(name string) (string, error) {
	base, err := runtimeIncludePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, name), nil
}

func runtimeSourcePaths() ([]string, error) {
	base, err := runtimeIncludePath()
	if err != nil {
		return nil, err
	}
	return []string{
		filepath.Join(base, "jayess_runtime.c"),
		filepath.Join(base, "jayess_runtime_bigint.c"),
		filepath.Join(base, "jayess_runtime_collections.c"),
		filepath.Join(base, "jayess_runtime_crypto.c"),
		filepath.Join(base, "jayess_runtime_errors.c"),
		filepath.Join(base, "jayess_runtime_fs.c"),
		filepath.Join(base, "jayess_runtime_process.c"),
		filepath.Join(base, "jayess_runtime_streams.c"),
		filepath.Join(base, "jayess_runtime_strings.c"),
		filepath.Join(base, "jayess_runtime_typed_arrays.c"),
		filepath.Join(base, "jayess_runtime_values.c"),
	}, nil
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
