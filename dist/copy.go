package dist

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func copyLLVMTools(buildDir string, outputDir string, tools []string) ([]string, []string, error) {
	sourceDir := llvmBuildBinDir(buildDir)
	var copied []string
	var diagnostics []string
	for _, tool := range tools {
		sourcePath := filepath.Join(sourceDir, tool)
		if _, err := os.Stat(sourcePath); err != nil {
			diagnostics = append(diagnostics, "missing LLVM tool: "+sourcePath)
			continue
		}
		if err := copyFile(sourcePath, filepath.Join(outputDir, tool), 0o755); err != nil {
			return copied, diagnostics, err
		}
		copied = append(copied, tool)
	}
	return copied, diagnostics, nil
}

func copyLLVMLibraries(buildDir string, outputDir string) ([]string, error) {
	sourceDir := llvmBuildLibDir(buildDir)
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var copied []string
	for _, entry := range entries {
		if entry.IsDir() || !isRuntimeLLVMLibrary(entry.Name()) {
			continue
		}
		sourcePath := filepath.Join(sourceDir, entry.Name())
		if err := copyFile(sourcePath, filepath.Join(outputDir, entry.Name()), 0o644); err != nil {
			return copied, err
		}
		copied = append(copied, entry.Name())
	}
	return copied, nil
}

func isRuntimeLLVMLibrary(name string) bool {
	return strings.HasPrefix(name, "libLLVM.") ||
		strings.HasPrefix(name, "libLLVM-") ||
		strings.HasPrefix(name, "libclang-cpp.") ||
		strings.HasPrefix(name, "libclang-cpp-") ||
		strings.HasPrefix(name, "LLVM-C.") ||
		strings.HasPrefix(name, "LLVM.") ||
		strings.HasPrefix(name, "clang-cpp.")
}

func copyFile(sourcePath string, outputPath string, mode os.FileMode) error {
	if err := os.MkdirAll(parentDir(outputPath), 0o755); err != nil {
		return err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	output, err := os.OpenFile(outputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, source); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

func parentDir(path string) string {
	dir := filepath.Dir(path)
	if dir == "." {
		return ""
	}
	return dir
}
