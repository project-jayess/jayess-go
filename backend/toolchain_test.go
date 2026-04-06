package backend

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"jayess-go/compiler"
	"jayess-go/target"
)

func TestBuildExecutableRunsCompiledProgram(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  console.log("hello native");
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "hello-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "hello native") {
		t.Fatalf("expected program output to contain hello native, got: %s", string(out))
	}
}

func TestBuildExecutablePassesRuntimeArgs(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  console.log(process.argv());
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "args-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath, "kimchi", "jjigae")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "kimchi") || !strings.Contains(text, "jjigae") {
		t.Fatalf("expected argv output to contain runtime args, got: %s", text)
	}
}

func TestBuildExecutableSupportsFsAndPathRuntimeSurface(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  fs.mkdir("tmp", { recursive: true });
  var file = path.join("tmp", "note.txt");
  fs.writeFile(file, "kimchi");
  console.log(path.basename(file));
  console.log(path.extname(file));
  console.log(fs.exists(file));
  console.log(fs.readFile(file));
  console.log(fs.stat(file).isFile);
  console.log(fs.readDir("tmp")[0].name);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "fs-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"note.txt", ".txt", "true", "kimchi"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected fs/path output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsNativeWrapperImports(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	nativeDir := filepath.Join(workdir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	nativeSource := `#include "jayess_runtime.h"

jayess_value *jayess_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_value_to_number(a) + jayess_value_to_number(b));
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(nativeSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { jayess_add } from "./native/math.c";

function main(args) {
  console.log(jayess_add(3, 4));
  return 0;
}
`
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "ffi-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "7") {
		t.Fatalf("expected native wrapper output to contain 7, got: %s", string(out))
	}
}

func TestBuildExecutableSupportsPackageImports(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	pkgDir := filepath.Join(workdir, "node_modules", "@demo", "math")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
export function add(a, b) {
  return a + b;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { add } from "@demo/math";

function main(args) {
  console.log(add(5, 6));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "pkg-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "11") {
		t.Fatalf("expected package-import output to contain 11, got: %s", string(out))
	}
}

func TestBuildExecutableSupportsPathHelperEdgeCases(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	result, err := compiler.Compile(`
function main(args) {
  console.log(path.resolve("a", "..", "b"));
  console.log(path.relative("tmp", path.join("tmp", "nested", "file.txt")));
  console.log(path.format(path.parse(path.join("tmp", "nested", "file.txt"))));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "path-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}

	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(out), "\r\n", "\n")), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected three output lines, got: %q", string(out))
	}

	expectedResolve := filepath.Clean(filepath.Join(workdir, "b"))
	expectedRelative := filepath.Join("nested", "file.txt")
	expectedFormat := filepath.Join("tmp", "nested", "file.txt")

	if lines[0] != expectedResolve {
		t.Fatalf("expected resolved path %q, got %q", expectedResolve, lines[0])
	}
	if lines[1] != expectedRelative {
		t.Fatalf("expected relative path %q, got %q", expectedRelative, lines[1])
	}
	if lines[2] != expectedFormat {
		t.Fatalf("expected formatted path %q, got %q", expectedFormat, lines[2])
	}
}

func TestBuildObjectSupportsConfiguredCrossTargets(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping cross-target build test: %v", err)
	}

	source := `
function main(args) {
  console.log("cross");
  return 0;
}
`
	for _, targetName := range []string{"windows-x64", "linux-x64", "darwin-arm64"} {
		t.Run(targetName, func(t *testing.T) {
			triple, err := target.FromName(targetName)
			if err != nil {
				t.Fatalf("FromName returned error: %v", err)
			}
			result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
			if err != nil {
				t.Fatalf("Compile returned error: %v", err)
			}
			outputPath := filepath.Join(t.TempDir(), targetName+".o")
			if err := tc.BuildObject(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
				t.Skipf("cross-target object build unavailable for %s: %v", targetName, err)
			}
			if info, err := os.Stat(outputPath); err != nil || info.IsDir() {
				t.Fatalf("expected built object file for %s, got err=%v", targetName, err)
			}
		})
	}
}

func TestBuildExecutableSupportsProcessPathAndRecursiveFsEdgeCases(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	result, err := compiler.Compile(`
function main(args) {
  fs.mkdir(path.join("tmp", "a", "b"), { recursive: true });
  fs.writeFile(path.join("tmp", "a", "b", "note.txt"), "kimchi");
  fs.copyDir("tmp", "copy");
  console.log(process.arch());
  console.log(path.sep);
  console.log(path.delimiter);
  console.log(fs.readFile(path.join("copy", "a", "b", "note.txt"), "utf8"));
  console.log(fs.readFile("missing.txt"));
  console.log(fs.stat("missing.txt"));
  console.log(fs.remove("copy", { recursive: true }));
  console.log(fs.exists("copy"));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "stdlib-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}

	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(out), "\r\n", "\n")), "\n")
	if len(lines) < 8 {
		t.Fatalf("expected at least eight output lines, got %q", string(out))
	}
	if lines[0] == "" {
		t.Fatalf("expected process.arch output, got %q", lines[0])
	}
	if lines[1] != string(filepath.Separator) {
		t.Fatalf("expected path.sep %q, got %q", string(filepath.Separator), lines[1])
	}
	expectedDelimiter := ":"
	if runtime.GOOS == "windows" {
		expectedDelimiter = ";"
	}
	if lines[2] != expectedDelimiter {
		t.Fatalf("expected path.delimiter %q, got %q", expectedDelimiter, lines[2])
	}
	if lines[3] != "kimchi" {
		t.Fatalf("expected copied file contents, got %q", lines[3])
	}
	if lines[4] != "undefined" {
		t.Fatalf("expected missing file read to return undefined, got %q", lines[4])
	}
	if lines[5] != "undefined" {
		t.Fatalf("expected missing file stat to return undefined, got %q", lines[5])
	}
	if lines[6] != "true" {
		t.Fatalf("expected recursive remove success, got %q", lines[6])
	}
	if lines[7] != "false" {
		t.Fatalf("expected removed directory to be absent, got %q", lines[7])
	}
}

func nativeOutputPath(dir, name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(dir, name+".exe")
	}
	return filepath.Join(dir, name)
}
