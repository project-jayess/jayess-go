package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIEmitsObjectFileWithToolchain(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-object-*")
	fakeBin := installFakeSharedToolchain(t, root)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	input := filepath.Join(dir, "object_app.js")
	output := filepath.Join(dir, "object_app.o")
	if err := os.WriteFile(input, []byte("function main() { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	runJayessCLI(t, root, "--target=linux-x64", "--emit=obj", "-o", output, input)
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read object output: %v", err)
	}
	text := string(content)
	for _, want := range []string{
		"fake shared library",
		"-target x86_64-pc-linux-gnu",
		"-c",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected object output to contain %q, got:\n%s", want, text)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "temp", "jayess-build", "object_app.ll")); err != nil {
		t.Fatalf("expected temporary LLVM IR file: %v", err)
	}
}

func TestCLIReportsMissingObjectToolBeforeWritingTempIR(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-missing-object-tool-*")
	emptyPath := cliTempDir(t, root, "empty-object-path-*")
	input := filepath.Join(dir, "missing_object_tool.js")
	output := filepath.Join(dir, "missing_object_tool.o")
	if err := os.WriteFile(input, []byte("function main() { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("find go executable: %v", err)
	}
	command := exec.Command(goPath, "run", "./cmd/jayess", "--target=linux-x64", "--emit=obj", "-o", output, input)
	command.Dir = root
	command.Env = append(os.Environ(), "PATH="+emptyPath)
	result, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("expected CLI to fail without clang, got output:\n%s", string(result))
	}
	if !strings.Contains(string(result), `missing toolchain tool "clang" for linux-x64`) {
		t.Fatalf("expected missing tool diagnostic, got:\n%s", string(result))
	}
	if _, err := os.Stat(filepath.Join(root, "temp", "jayess-build", "missing_object_tool.ll")); !os.IsNotExist(err) {
		t.Fatalf("expected no temporary IR after missing tool preflight, stat error: %v", err)
	}
}

func TestCLIEmitsObjectFileWithInternalLLVMWhenAvailable(t *testing.T) {
	root := cliRepoRoot(t)
	if _, err := os.Stat(filepath.Join(root, "refs", "llvm-project", "build", "lib", "libLLVM.so")); err != nil {
		t.Skipf("repo-local LLVM build is not available: %v", err)
	}
	dir := cliTempDir(t, root, "cli-internal-object-*")
	compilerPath := filepath.Join(dir, "jayess")
	if runtime.GOOS == "windows" {
		compilerPath += ".exe"
	}
	build := exec.Command("go", "build", "-tags", "jayess_llvmc", "-o", compilerPath, "./cmd/jayess")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build internal LLVM jayess CLI: %v\n%s", err, string(output))
	}
	input := filepath.Join(dir, "internal_object.js")
	output := filepath.Join(dir, "internal_object.o")
	if err := os.WriteFile(input, []byte("function main() { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	_ = os.Remove(filepath.Join(root, "temp", "jayess-build", "internal_object.ll"))
	command := exec.Command(compilerPath, "--target=linux-x64", "--emit=obj", "-o", output, input)
	command.Dir = root
	command.Env = append(os.Environ(), "PATH="+cliTempDir(t, root, "empty-internal-object-path-*"))
	result, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("internal LLVM jayess CLI failed: %v\n%s", err, string(result))
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("stat internal object output: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected non-empty internal object output")
	}
	if _, err := os.Stat(filepath.Join(root, "temp", "jayess-build", "internal_object.ll")); !os.IsNotExist(err) {
		t.Fatalf("expected internal object emission to avoid temporary IR, stat error: %v", err)
	}
}
