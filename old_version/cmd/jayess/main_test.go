package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"jayess-go/backend"
	"jayess-go/compiler"
)

func TestFormatDiagnosticWithSnippet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.js")
	source := "function main(args) {\n  print(\"hello\");\n}\n"
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	text := formatDiagnosticWithSnippet(compiler.Diagnostic{
		Severity: "warning",
		Category: "deprecation",
		Code:     "JY001",
		File:     path,
		Line:     2,
		Column:   3,
		Message:  "deprecated",
	})

	if !strings.Contains(text, path+":2:3: warning[JY001]/deprecation: deprecated") {
		t.Fatalf("expected formatted location, got: %s", text)
	}
	if !strings.Contains(text, "  print(\"hello\");") {
		t.Fatalf("expected source snippet, got: %s", text)
	}
	if !strings.Contains(text, "\n  ^") {
		t.Fatalf("expected caret line, got: %s", text)
	}
}

func TestFormatCompileErrorWithSnippet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.js")
	source := "function main(args) {\n  return missing;\n}\n"
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	text := formatCompileErrorWithSnippet(path, os.ErrInvalid)
	if text != os.ErrInvalid.Error() {
		t.Fatalf("expected non-located errors to pass through, got: %s", text)
	}

	located := formatCompileErrorWithSnippet(path, &compiler.CompileError{
		Diagnostic: compiler.Diagnostic{
			Severity: "error",
			Category: "semantic",
			Code:     "JY200",
			File:     path,
			Line:     2,
			Column:   10,
			Message:  "unknown identifier missing",
		},
	})
	if !strings.Contains(located, path+":2:10: error[JY200]/semantic: unknown identifier missing") {
		t.Fatalf("expected located error formatting, got: %s", located)
	}
	if !strings.Contains(located, "  return missing;") {
		t.Fatalf("expected source snippet, got: %s", located)
	}
	if !strings.Contains(located, "\n         ^") {
		t.Fatalf("expected caret line, got: %s", located)
	}
}

func TestPrintWarningsAcceptsEmptyAndNonEmptyLists(t *testing.T) {
	warnings := []compiler.Diagnostic{{
		Severity: "warning",
		Category: "deprecation",
		Code:     "JY001",
		Message:  "deprecated",
	}}

	printWarnings(nil)
	printWarnings(warnings)
}

func TestStringListFlagAcceptsRepeatedAndCommaSeparatedValues(t *testing.T) {
	var values stringListFlag
	if err := values.Set("deprecation, compatibility"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if err := values.Set("style"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if got := values.String(); got != "deprecation,compatibility,style" {
		t.Fatalf("unexpected String output: %q", got)
	}
	want := []string{"deprecation", "compatibility", "style"}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("expected value %d to be %q, got %q", i, want[i], values[i])
		}
	}
}

func TestParseCLICompileDefaultsAndFlags(t *testing.T) {
	cfg, err := parseCLI([]string{"--target=windows-x64", "--emit=llvm", "-o", "build/out.ll", "sample.js"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if cfg.mode != "compile" {
		t.Fatalf("expected compile mode, got %q", cfg.mode)
	}
	if cfg.targetName != "windows-x64" || cfg.emit != "llvm" || cfg.output != "build/out.ll" || cfg.inputPath != "sample.js" {
		t.Fatalf("unexpected parsed compile config: %#v", cfg)
	}
}

func TestParseCLICompileSupportsObjectEmit(t *testing.T) {
	cfg, err := parseCLI([]string{"--target=linux-x64", "--emit=obj", "-o", "build/out.o", "sample.js"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if cfg.emit != "obj" || cfg.output != "build/out.o" || cfg.inputPath != "sample.js" {
		t.Fatalf("unexpected parsed object config: %#v", cfg)
	}
}

func TestParseCLICompileSupportsBitcodeEmit(t *testing.T) {
	cfg, err := parseCLI([]string{"--target=linux-x64", "--emit=bc", "-o", "build/out.bc", "sample.js"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if cfg.emit != "bc" || cfg.output != "build/out.bc" || cfg.inputPath != "sample.js" {
		t.Fatalf("unexpected parsed bitcode config: %#v", cfg)
	}
}

func TestParseCLICompileSupportsStaticLibraryEmit(t *testing.T) {
	cfg, err := parseCLI([]string{"--target=linux-x64", "--emit=lib", "-o", "build/libsample.a", "sample.js"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if cfg.emit != "lib" || cfg.output != "build/libsample.a" || cfg.inputPath != "sample.js" {
		t.Fatalf("unexpected parsed static library config: %#v", cfg)
	}
}

func TestParseCLICompileSupportsSharedLibraryEmit(t *testing.T) {
	cfg, err := parseCLI([]string{"--target=linux-x64", "--emit=shared", "-o", "build/libsample.so", "sample.js"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if cfg.emit != "shared" || cfg.output != "build/libsample.so" || cfg.inputPath != "sample.js" {
		t.Fatalf("unexpected parsed shared library config: %#v", cfg)
	}
}

func TestParseCLICompileSupportsOptimizationFlag(t *testing.T) {
	cfg, err := parseCLI([]string{"--target=linux-x64", "--emit=exe", "--opt=O2", "sample.js"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if cfg.optimizationLevel != "O2" {
		t.Fatalf("expected optimization level O2, got %#v", cfg)
	}
}

func TestParseCLICompileSupportsTargetCodegenFlags(t *testing.T) {
	cfg, err := parseCLI([]string{
		"--target=linux-x64",
		"--cpu=native",
		"--feature=+sse2,-avx",
		"--feature=+aes",
		"--reloc=pic",
		"--code-model=small",
		"sample.js",
	})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if cfg.targetCPU != "native" {
		t.Fatalf("expected target CPU native, got %#v", cfg)
	}
	if got := strings.Join(cfg.targetFeatures, ","); got != "+sse2,-avx,+aes" {
		t.Fatalf("unexpected target features: %q", got)
	}
	if cfg.relocationModel != "pic" {
		t.Fatalf("expected relocation model pic, got %#v", cfg)
	}
	if cfg.codeModel != "small" {
		t.Fatalf("expected code model small, got %#v", cfg)
	}
}

func TestParseCLIRunCommandCollectsProgramArgs(t *testing.T) {
	cfg, err := parseCLI([]string{"run", "--target=host", "sample.js", "one", "two"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if cfg.mode != "run" {
		t.Fatalf("expected run mode, got %q", cfg.mode)
	}
	if cfg.inputPath != "sample.js" {
		t.Fatalf("expected input path sample.js, got %q", cfg.inputPath)
	}
	if got := strings.Join(cfg.programArgs, ","); got != "one,two" {
		t.Fatalf("unexpected program args: %q", got)
	}
}

func TestUsageForMode(t *testing.T) {
	if !strings.Contains(usageForMode("compile"), "--emit=llvm|bc|obj|lib|shared|exe") || !strings.Contains(usageForMode("compile"), "--opt=O0|O1|O2|O3|Oz") || !strings.Contains(usageForMode("compile"), "--cpu=<name>") || !strings.Contains(usageForMode("compile"), "--feature=<flag>") || !strings.Contains(usageForMode("compile"), "--reloc=pic|pie|static") || !strings.Contains(usageForMode("compile"), "--code-model=small|medium|large|kernel") {
		t.Fatalf("expected compile usage to mention emit")
	}
	if !strings.Contains(usageForMode("run"), "jayess run") || !strings.Contains(usageForMode("run"), "--opt=O0|O1|O2|O3|Oz") || !strings.Contains(usageForMode("run"), "--cpu=<name>") || !strings.Contains(usageForMode("run"), "--feature=<flag>") || !strings.Contains(usageForMode("run"), "--reloc=pic|pie|static") || !strings.Contains(usageForMode("run"), "--code-model=small|medium|large|kernel") {
		t.Fatalf("expected run usage to mention run command")
	}
	if !strings.Contains(usageForMode("test"), "jayess test") || !strings.Contains(usageForMode("test"), "--cpu=<name>") || !strings.Contains(usageForMode("test"), "--feature=<flag>") || !strings.Contains(usageForMode("test"), "--reloc=pic|pie|static") || !strings.Contains(usageForMode("test"), "--code-model=small|medium|large|kernel") {
		t.Fatalf("expected test usage to mention test command")
	}
}

func TestIsSupportedOptimizationLevel(t *testing.T) {
	for _, level := range []string{"O0", "O1", "O2", "O3", "Oz"} {
		if !isSupportedOptimizationLevel(level) {
			t.Fatalf("expected %s to be supported", level)
		}
	}
	if isSupportedOptimizationLevel("Ofast") {
		t.Fatalf("expected Ofast to be unsupported")
	}
}

func TestIsSupportedRelocationModel(t *testing.T) {
	for _, model := range []string{"pic", "pie", "static"} {
		if !isSupportedRelocationModel(model) {
			t.Fatalf("expected %s to be supported", model)
		}
	}
	if isSupportedRelocationModel("dynamic-no-pic") {
		t.Fatalf("expected dynamic-no-pic to be unsupported")
	}
}

func TestIsSupportedCodeModel(t *testing.T) {
	for _, model := range []string{"small", "medium", "large", "kernel"} {
		if !isSupportedCodeModel(model) {
			t.Fatalf("expected %s to be supported", model)
		}
	}
	if isSupportedCodeModel("tiny") {
		t.Fatalf("expected tiny to be unsupported")
	}
}

func TestCompileCLIRejectsUnsupportedRelocationModel(t *testing.T) {
	_, err := compileCLI(cliConfig{
		mode:            "compile",
		targetName:      "linux-x64",
		inputPath:       "sample.js",
		relocationModel: "dynamic-no-pic",
	}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), `unsupported relocation model "dynamic-no-pic"`) {
		t.Fatalf("expected unsupported relocation model error, got: %v", err)
	}
}

func TestCompileCLIRejectsUnsupportedCodeModel(t *testing.T) {
	_, err := compileCLI(cliConfig{
		mode:       "compile",
		targetName: "linux-x64",
		inputPath:  "sample.js",
		codeModel:  "tiny",
	}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), `unsupported code model "tiny"`) {
		t.Fatalf("expected unsupported code model error, got: %v", err)
	}
}

func TestDefaultOutputPathSupportsObjectEmit(t *testing.T) {
	if got := defaultOutputPath("sample.js", "obj"); got != filepath.Join("build", "sample.o") {
		t.Fatalf("unexpected object output path: %q", got)
	}
}

func TestDefaultOutputPathSupportsBitcodeEmit(t *testing.T) {
	if got := defaultOutputPath("sample.js", "bc"); got != filepath.Join("build", "sample.bc") {
		t.Fatalf("unexpected bitcode output path: %q", got)
	}
}

func TestDefaultOutputPathSupportsStaticLibraryEmit(t *testing.T) {
	if got := defaultOutputPath("sample.js", "lib"); got != filepath.Join("build", "libsample.a") {
		t.Fatalf("unexpected static library output path: %q", got)
	}
}

func TestDefaultOutputPathSupportsSharedLibraryEmit(t *testing.T) {
	var want string
	switch runtime.GOOS {
	case "windows":
		want = filepath.Join("build", "sample.dll")
	case "darwin":
		want = filepath.Join("build", "libsample.dylib")
	default:
		want = filepath.Join("build", "libsample.so")
	}
	if got := defaultOutputPath("sample.js", "shared"); got != want {
		t.Fatalf("unexpected shared library output path: %q", got)
	}
}

func TestParseCLIInitCommand(t *testing.T) {
	cfg, err := parseCLI([]string{"init", "demo-app"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if cfg.mode != "init" {
		t.Fatalf("expected init mode, got %q", cfg.mode)
	}
	if cfg.initPath != "demo-app" {
		t.Fatalf("expected init path demo-app, got %q", cfg.initPath)
	}
}

func TestParseCLITestCommandDefaultsToCurrentDirectory(t *testing.T) {
	cfg, err := parseCLI([]string{"test"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if cfg.mode != "test" {
		t.Fatalf("expected test mode, got %q", cfg.mode)
	}
	if cfg.inputPath != "." {
		t.Fatalf("expected test input path ., got %q", cfg.inputPath)
	}
}

func TestDiscoverTestFilesFindsNestedTestFilesAndSkipsBuildLikeDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "alpha.test.js"), []byte("function main(args) { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nested", "beta.test.js"), []byte("function main(args) { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "node_modules", "skip.test.js"), []byte("function main(args) { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	files, err := discoverTestFiles(dir)
	if err != nil {
		t.Fatalf("discoverTestFiles returned error: %v", err)
	}
	if got := strings.Join(files, ","); !strings.Contains(got, "alpha.test.js") || !strings.Contains(got, "beta.test.js") || strings.Contains(got, "skip.test.js") {
		t.Fatalf("unexpected discovered files: %q", got)
	}
}

func TestRunCLIInitCreatesPackageFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo-app")
	var stderr bytes.Buffer
	if err := runCLI([]string{"init", dir}, &stderr); err != nil {
		t.Fatalf("runCLI returned error: %v", err)
	}
	packageJSON, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatalf("ReadFile package.json returned error: %v", err)
	}
	mainSource, err := os.ReadFile(filepath.Join(dir, "main.js"))
	if err != nil {
		t.Fatalf("ReadFile main.js returned error: %v", err)
	}
	if !strings.Contains(string(packageJSON), "\"name\": \"demo-app\"") {
		t.Fatalf("unexpected package.json contents: %s", packageJSON)
	}
	if !strings.Contains(string(mainSource), "function main(args)") {
		t.Fatalf("unexpected main.js contents: %s", mainSource)
	}
}

func TestRunCLIInitRejectsExistingPackage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	var stderr bytes.Buffer
	err := runCLI([]string{"init", dir}, &stderr)
	if err == nil {
		t.Fatalf("expected init error")
	}
	if !strings.Contains(err.Error(), "package.json already exists") {
		t.Fatalf("unexpected init error: %v", err)
	}
}

func TestRunCLITestExecutesPassingTestFiles(t *testing.T) {
	if _, err := backend.DetectToolchain(); err != nil {
		t.Skipf("skipping native CLI test runner test: %v", err)
	}

	dir := t.TempDir()
	testFile := filepath.Join(dir, "sample.test.js")
	if err := os.WriteFile(testFile, []byte("function main(args) {\n  console.log(\"ok\");\n  return 0;\n}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stderr bytes.Buffer
	if err := runCLI([]string{"test", dir}, &stderr); err != nil {
		t.Fatalf("runCLI returned error: %v", err)
	}
	text := stderr.String()
	if !strings.Contains(text, "PASS "+testFile) {
		t.Fatalf("expected test runner PASS output, got: %s", text)
	}
	if !strings.Contains(text, "PASS 1 test files") {
		t.Fatalf("expected test runner summary, got: %s", text)
	}
}

func TestRunCLICompileEmitsObjectFile(t *testing.T) {
	if _, err := backend.DetectToolchain(); err != nil {
		t.Skipf("skipping object emit CLI test: %v", err)
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "sample.js")
	output := filepath.Join(dir, "sample.o")
	if err := os.WriteFile(input, []byte("function main(args) {\n  return 0;\n}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	var stderr bytes.Buffer
	if err := runCLI([]string{"--target=host", "--emit=obj", "-o", output, input}, &stderr); err != nil {
		t.Fatalf("runCLI returned error: %v", err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if info.IsDir() || info.Size() == 0 {
		t.Fatalf("expected non-empty object file, got size=%d", info.Size())
	}
}

func TestRunCLICompileEmitsBitcodeFile(t *testing.T) {
	tc, err := backend.DetectToolchain()
	if err != nil {
		t.Skipf("skipping bitcode emit CLI test: %v", err)
	}
	if tc.LLVMAsPath == "" {
		t.Skip("skipping bitcode emit CLI test: llvm-as unavailable")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "sample.js")
	output := filepath.Join(dir, "sample.bc")
	if err := os.WriteFile(input, []byte("function main(args) {\n  return 0;\n}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	var stderr bytes.Buffer
	if err := runCLI([]string{"--target=host", "--emit=bc", "-o", output, input}, &stderr); err != nil {
		t.Fatalf("runCLI returned error: %v", err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if info.IsDir() || info.Size() == 0 {
		t.Fatalf("expected non-empty bitcode file, got size=%d", info.Size())
	}
}

func TestRunCLICompileEmitsStaticLibraryFile(t *testing.T) {
	tc, err := backend.DetectToolchain()
	if err != nil {
		t.Skipf("skipping static library emit CLI test: %v", err)
	}
	if tc.ARPath == "" {
		t.Skip("skipping static library emit CLI test: ar unavailable")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "sample.js")
	output := filepath.Join(dir, "libsample.a")
	if err := os.WriteFile(input, []byte("function main(args) {\n  return 0;\n}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	var stderr bytes.Buffer
	if err := runCLI([]string{"--target=host", "--emit=lib", "-o", output, input}, &stderr); err != nil {
		t.Fatalf("runCLI returned error: %v", err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if info.IsDir() || info.Size() == 0 {
		t.Fatalf("expected non-empty static library file, got size=%d", info.Size())
	}
}

func TestRunCLICompileEmitsSharedLibraryFile(t *testing.T) {
	if _, err := backend.DetectToolchain(); err != nil {
		t.Skipf("skipping shared library emit CLI test: %v", err)
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "sample.js")
	output := defaultOutputPath(input, "shared")
	if err := os.WriteFile(input, []byte("function main(args) {\n  return 0;\n}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	var stderr bytes.Buffer
	if err := runCLI([]string{"--target=host", "--emit=shared", "-o", output, input}, &stderr); err != nil {
		t.Fatalf("runCLI returned error: %v", err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if info.IsDir() || info.Size() == 0 {
		t.Fatalf("expected non-empty shared library file, got size=%d", info.Size())
	}
}

func TestRunCLICompileRejectsUnsupportedOptimizationLevel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "sample.js")
	if err := os.WriteFile(input, []byte("function main(args) {\n  return 0;\n}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	var stderr bytes.Buffer
	err := runCLI([]string{"--target=host", "--emit=obj", "--opt=Ofast", input}, &stderr)
	if err == nil || !strings.Contains(err.Error(), "unsupported optimization level") {
		t.Fatalf("expected unsupported optimization level error, got %v", err)
	}
}
