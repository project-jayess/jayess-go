package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIEmitsLLVMIR(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-llvm-*")
	input := filepath.Join(dir, "hello.js")
	output := filepath.Join(dir, "hello.ll")
	if err := os.WriteFile(input, []byte("function main() { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	result := runJayessCLI(t, root, "--target=linux-x64", "--emit=llvm", "-o", output, input)
	if result != "" {
		t.Fatalf("expected no CLI output, got %q", result)
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read LLVM output: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, `target triple = "x86_64-pc-linux-gnu"`) {
		t.Fatalf("expected target triple in LLVM IR, got:\n%s", text)
	}
	if !strings.Contains(text, "define i32 @main()") {
		t.Fatalf("expected main function in LLVM IR, got:\n%s", text)
	}
}

func TestCLIEmitsSharedLibraryWithToolchain(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-shared-*")
	fakeBin := installFakeSharedToolchain(t, root)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	input := filepath.Join(dir, "native_app.js")
	output := filepath.Join(dir, "libnative_app.so")
	if err := os.WriteFile(input, []byte("function main() { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	runJayessCLI(t, root, "compile", "--target=linux-x64", "--emit=shared", "-o", output, input)
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read shared library output: %v", err)
	}
	text := string(content)
	for _, want := range []string{
		"fake shared library",
		"-target x86_64-pc-linux-gnu",
		"-shared",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected shared library output to contain %q, got:\n%s", want, text)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "temp", "jayess-build", "libnative_app.ll")); err != nil {
		t.Fatalf("expected temporary LLVM IR file: %v", err)
	}
}

func TestCLIUsesTargetSharedLibraryDefaultName(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-shared-name-*")
	fakeBin := installFakeSharedToolchain(t, root)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	input := filepath.Join(dir, "plugin.js")
	if err := os.WriteFile(input, []byte("function main() { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	command := exec.Command("go", "run", "./cmd/jayess", "--target=macos-arm64", "--emit=shared", input)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("jayess CLI failed: %v\n%s", err, string(output))
	}
	planPath := filepath.Join(root, "build", "libplugin.dylib")
	t.Cleanup(func() { _ = os.Remove(planPath) })
	content, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("read default shared library output: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "-target arm64-apple-darwin") || !strings.Contains(text, "-dynamiclib") {
		t.Fatalf("expected macOS shared-library command output, got:\n%s", text)
	}
}

func TestCLIReportsMissingSharedLibraryToolBeforeWritingTempIR(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-missing-tool-*")
	emptyPath := cliTempDir(t, root, "empty-path-*")
	input := filepath.Join(dir, "missing_tool.js")
	output := filepath.Join(dir, "libmissing_tool.so")
	if err := os.WriteFile(input, []byte("function main() { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("find go executable: %v", err)
	}
	command := exec.Command(goPath, "run", "./cmd/jayess", "--target=linux-x64", "--emit=shared", "-o", output, input)
	command.Dir = root
	command.Env = append(os.Environ(), "PATH="+emptyPath)
	result, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("expected CLI to fail without LLVM tools, got output:\n%s", string(result))
	}
	if !strings.Contains(string(result), `missing toolchain tool "clang" for linux-x64`) {
		t.Fatalf("expected missing tool diagnostic, got:\n%s", string(result))
	}
	if _, err := os.Stat(filepath.Join(root, "temp", "jayess-build", "libmissing_tool.ll")); !os.IsNotExist(err) {
		t.Fatalf("expected no temporary IR after missing tool preflight, stat error: %v", err)
	}
}

func TestCLIUsesJayessToolchainBeforePATH(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-env-toolchain-*")
	toolchainRoot := cliTempDir(t, root, "jayess-toolchain-*")
	toolchainBin := filepath.Join(toolchainRoot, "linux-x64", "bin")
	if err := os.MkdirAll(toolchainBin, 0o755); err != nil {
		t.Fatalf("create toolchain bin: %v", err)
	}
	installFakeSharedToolchainAt(t, toolchainBin)
	t.Setenv("JAYESS_TOOLCHAIN", toolchainRoot)
	input := filepath.Join(dir, "toolchain_app.js")
	output := filepath.Join(dir, "libtoolchain_app.so")
	if err := os.WriteFile(input, []byte("function main() { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("find go executable: %v", err)
	}
	command := exec.Command(goPath, "run", "./cmd/jayess", "--target=linux-x64", "--emit=shared", "-o", output, input)
	command.Dir = root
	command.Env = append(os.Environ(), "PATH="+filepath.Dir(goPath), "JAYESS_TOOLCHAIN="+toolchainRoot)
	result, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("jayess CLI failed: %v\n%s", err, string(result))
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read shared library output: %v", err)
	}
	if !strings.Contains(string(content), "fake shared library") {
		t.Fatalf("expected fake toolchain output, got:\n%s", string(content))
	}
}

func TestCLIUsesBundledClangBesideCompiler(t *testing.T) {
	root := cliRepoRoot(t)
	releaseDir := cliTempDir(t, root, "jayess-release-*")
	compilerPath := filepath.Join(releaseDir, "jayess")
	if runtime.GOOS == "windows" {
		compilerPath += ".exe"
	}
	build := exec.Command("go", "build", "-o", compilerPath, "./cmd/jayess")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build jayess CLI: %v\n%s", err, string(output))
	}

	toolchainBin := filepath.Join(releaseDir, "tools", "linux-x64", "bin")
	if err := os.MkdirAll(toolchainBin, 0o755); err != nil {
		t.Fatalf("create bundled toolchain bin: %v", err)
	}
	installFakeSharedToolchainAt(t, toolchainBin)

	input := filepath.Join(releaseDir, "bundled_app.js")
	output := filepath.Join(releaseDir, "libbundled_app.so")
	if err := os.WriteFile(input, []byte("function main() { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	command := exec.Command(compilerPath, "--target=linux-x64", "--emit=shared", "-o", output, input)
	command.Dir = root
	command.Env = append(os.Environ(), "PATH="+cliTempDir(t, root, "empty-bundled-path-*"), "JAYESS_TOOLCHAIN=")
	result, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("bundled jayess CLI failed: %v\n%s", err, string(result))
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read bundled shared library output: %v", err)
	}
	if !strings.Contains(string(content), "fake shared library") {
		t.Fatalf("expected bundled clang output, got:\n%s", string(content))
	}
}
