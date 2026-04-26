package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"jayess-go/backend"
	"jayess-go/compiler"
	"jayess-go/target"
)

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	if value == "" {
		return nil
	}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			*f = append(*f, part)
		}
	}
	return nil
}

func main() {
	if err := runCLI(os.Args[1:], os.Stderr); err != nil {
		var diagnosticErr *cliDiagnosticError
		if errors.As(err, &diagnosticErr) {
			fmt.Fprintln(os.Stderr, formatCompileErrorWithSnippet(diagnosticErr.inputPath, diagnosticErr.err))
			os.Exit(1)
		}
		exitf("%v", err)
	}
}

type cliDiagnosticError struct {
	inputPath string
	err       error
}

func (e *cliDiagnosticError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

type cliConfig struct {
	mode                     string
	emit                     string
	optimizationLevel        string
	targetCPU                string
	targetFeatures           stringListFlag
	relocationModel          string
	codeModel                string
	targetName               string
	output                   string
	warningPolicy            string
	allowedWarningCategories stringListFlag
	inputPath                string
	initPath                 string
	programArgs              []string
}

func runCLI(args []string, stderr io.Writer) error {
	cfg, err := parseCLI(args)
	if err != nil {
		return err
	}
	switch cfg.mode {
	case "compile":
		_, err := compileCLI(cfg, stderr)
		return err
	case "init":
		return initPackage(cfg)
	case "run":
		result, err := compileCLI(cfg, stderr)
		if err != nil {
			return err
		}
		hostTriple, err := target.DefaultTriple()
		if err != nil {
			return err
		}
		if result.targetTriple != hostTriple {
			return fmt.Errorf("run only supports the host target %s, got %s", hostTriple, result.targetTriple)
		}
		command := exec.Command(result.outputPath, cfg.programArgs...)
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				os.Exit(exitErr.ExitCode())
			}
			return fmt.Errorf("run executable: %w", err)
		}
		return nil
	case "test":
		return runTests(cfg, stderr)
	default:
		return fmt.Errorf("unsupported command %q", cfg.mode)
	}
}

func initPackage(cfg cliConfig) error {
	dir := cfg.initPath
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create project directory: %w", err)
	}
	name := filepath.Base(dir)
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "jayess-app"
	}
	packageJSONPath := filepath.Join(dir, "package.json")
	mainPath := filepath.Join(dir, "main.js")
	if _, err := os.Stat(packageJSONPath); err == nil {
		return fmt.Errorf("package.json already exists in %s", dir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat package.json: %w", err)
	}
	if _, err := os.Stat(mainPath); err == nil {
		return fmt.Errorf("main.js already exists in %s", dir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat main.js: %w", err)
	}
	packageJSON := fmt.Sprintf("{\n  \"name\": %q,\n  \"private\": true,\n  \"version\": \"0.1.0\"\n}\n", name)
	mainSource := "function main(args) {\n  console.log(\"Hello, Jayess\");\n  return 0;\n}\n"
	if err := os.WriteFile(packageJSONPath, []byte(packageJSON), 0o644); err != nil {
		return fmt.Errorf("write package.json: %w", err)
	}
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		return fmt.Errorf("write main.js: %w", err)
	}
	return nil
}

type compileResult struct {
	targetTriple string
	outputPath   string
}

func compileCLI(cfg cliConfig, stderr io.Writer) (*compileResult, error) {
	var emit string
	var targetName string
	var optimizationLevel string
	var targetCPU string
	var targetFeatures stringListFlag
	var relocationModel string
	var codeModel string
	var output string
	var warningPolicy string
	var allowedWarningCategories stringListFlag
	emit = cfg.emit
	targetName = cfg.targetName
	optimizationLevel = cfg.optimizationLevel
	targetCPU = cfg.targetCPU
	targetFeatures = cfg.targetFeatures
	relocationModel = cfg.relocationModel
	codeModel = cfg.codeModel
	output = cfg.output
	warningPolicy = cfg.warningPolicy
	allowedWarningCategories = cfg.allowedWarningCategories

	targetTriple, err := target.FromName(targetName)
	if err != nil {
		return nil, fmt.Errorf("resolve target: %v", err)
	}

	if emit == "" {
		if cfg.mode == "run" {
			emit = "exe"
		} else {
			emit = defaultEmitMode()
		}
	}
	if cfg.mode == "run" && emit != "exe" {
		return nil, fmt.Errorf("run requires --emit=exe")
	}
	if optimizationLevel == "" {
		optimizationLevel = "O0"
	}
	if !isSupportedOptimizationLevel(optimizationLevel) {
		return nil, fmt.Errorf("unsupported optimization level %q", optimizationLevel)
	}
	if relocationModel != "" && !isSupportedRelocationModel(relocationModel) {
		return nil, fmt.Errorf("unsupported relocation model %q", relocationModel)
	}
	if codeModel != "" && !isSupportedCodeModel(codeModel) {
		return nil, fmt.Errorf("unsupported code model %q", codeModel)
	}

	if output == "" {
		if cfg.mode == "run" {
			tempDir, err := os.MkdirTemp("", "jayess-run-*")
			if err != nil {
				return nil, fmt.Errorf("create temp output directory: %v", err)
			}
			name := strings.TrimSuffix(filepath.Base(cfg.inputPath), filepath.Ext(cfg.inputPath))
			if name == "" {
				name = "main"
			}
			if runtime.GOOS == "windows" {
				output = filepath.Join(tempDir, name+".exe")
			} else {
				output = filepath.Join(tempDir, name)
			}
		} else {
			output = defaultOutputPath(cfg.inputPath, emit)
		}
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return nil, fmt.Errorf("create output directory: %v", err)
	}

	opts := compiler.Options{
		TargetTriple:             targetTriple,
		WarningPolicy:            warningPolicy,
		AllowedWarningCategories: allowedWarningCategories,
		OptimizationLevel:        optimizationLevel,
		TargetCPU:                targetCPU,
		TargetFeatures:           targetFeatures,
		RelocationModel:          relocationModel,
		CodeModel:                codeModel,
	}
	result, err := compiler.CompilePath(cfg.inputPath, opts)
	if err != nil {
		return nil, &cliDiagnosticError{inputPath: cfg.inputPath, err: err}
	}
	printWarningsTo(stderr, result.Warnings)

	switch emit {
	case "llvm":
		if err := os.WriteFile(output, result.LLVMIR, 0o644); err != nil {
			return nil, fmt.Errorf("write output: %v", err)
		}
	case "bc":
		tc, err := backend.DetectToolchain()
		if err != nil {
			return nil, fmt.Errorf("detect LLVM toolchain: %v", err)
		}
		if err := tc.BuildBitcode(result, output); err != nil {
			return nil, err
		}
	case "obj":
		tc, err := backend.DetectToolchain()
		if err != nil {
			return nil, fmt.Errorf("detect LLVM toolchain: %v", err)
		}
		if err := tc.BuildObject(result, opts, output); err != nil {
			return nil, fmt.Errorf("build object: %v", err)
		}
	case "lib":
		tc, err := backend.DetectToolchain()
		if err != nil {
			return nil, fmt.Errorf("detect LLVM toolchain: %v", err)
		}
		if err := tc.BuildStaticLibrary(result, opts, output); err != nil {
			return nil, fmt.Errorf("build static library: %v", err)
		}
	case "shared":
		tc, err := backend.DetectToolchain()
		if err != nil {
			return nil, fmt.Errorf("detect LLVM toolchain: %v", err)
		}
		if err := tc.BuildSharedLibrary(result, opts, output); err != nil {
			return nil, fmt.Errorf("build shared library: %v", err)
		}
	case "exe":
		tc, err := backend.DetectToolchain()
		if err != nil {
			return nil, fmt.Errorf("detect LLVM toolchain: %v", err)
		}
		if err := tc.BuildExecutable(result, opts, output); err != nil {
			return nil, fmt.Errorf("build executable: %v", err)
		}
	default:
		return nil, fmt.Errorf("unsupported emit mode %q", emit)
	}
	return &compileResult{targetTriple: targetTriple, outputPath: output}, nil
}

func runTests(cfg cliConfig, stderr io.Writer) error {
	hostTriple, err := target.DefaultTriple()
	if err != nil {
		return err
	}
	targetTriple, err := target.FromName(cfg.targetName)
	if err != nil {
		return fmt.Errorf("resolve target: %v", err)
	}
	if targetTriple != hostTriple {
		return fmt.Errorf("test only supports the host target %s, got %s", hostTriple, targetTriple)
	}

	testFiles, err := discoverTestFiles(cfg.inputPath)
	if err != nil {
		return err
	}
	if len(testFiles) == 0 {
		return fmt.Errorf("no .test.js files found at %s", cfg.inputPath)
	}

	tc, err := backend.DetectToolchain()
	if err != nil {
		return fmt.Errorf("detect LLVM toolchain: %v", err)
	}

	opts := compiler.Options{
		TargetTriple:             hostTriple,
		WarningPolicy:            cfg.warningPolicy,
		AllowedWarningCategories: cfg.allowedWarningCategories,
	}
	passed := 0
	for _, testFile := range testFiles {
		tempDir, err := os.MkdirTemp("", "jayess-test-*")
		if err != nil {
			return fmt.Errorf("create test temp directory: %w", err)
		}
		name := strings.TrimSuffix(filepath.Base(testFile), filepath.Ext(testFile))
		if name == "" {
			name = "test"
		}
		outputPath := filepath.Join(tempDir, name)
		if runtime.GOOS == "windows" {
			outputPath += ".exe"
		}

		result, err := compiler.CompilePath(testFile, opts)
		if err != nil {
			_ = os.RemoveAll(tempDir)
			return &cliDiagnosticError{inputPath: testFile, err: err}
		}
		printWarningsTo(stderr, result.Warnings)
		if err := tc.BuildExecutable(result, opts, outputPath); err != nil {
			_ = os.RemoveAll(tempDir)
			return fmt.Errorf("build test %s: %v", testFile, err)
		}

		command := exec.Command(outputPath)
		command.Dir = filepath.Dir(testFile)
		output, err := command.CombinedOutput()
		if err != nil {
			fmt.Fprintf(stderr, "FAIL %s\n%s", testFile, string(output))
			_ = os.RemoveAll(tempDir)
			return fmt.Errorf("test failed: %s", testFile)
		}
		fmt.Fprintf(stderr, "PASS %s\n", testFile)
		passed++
		_ = os.RemoveAll(tempDir)
	}
	fmt.Fprintf(stderr, "PASS %d test files\n", passed)
	return nil
}

func discoverTestFiles(inputPath string) ([]string, error) {
	if inputPath == "" {
		inputPath = "."
	}
	info, err := os.Stat(inputPath)
	if err != nil {
		return nil, fmt.Errorf("stat test path: %w", err)
	}
	if !info.IsDir() {
		if strings.HasSuffix(inputPath, ".test.js") {
			return []string{inputPath}, nil
		}
		return nil, fmt.Errorf("test path must be a directory or .test.js file")
	}

	var files []string
	err = filepath.WalkDir(inputPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if path != inputPath && (name == "node_modules" || name == "build" || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".test.js") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk test path: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func printWarnings(warnings []compiler.Diagnostic) {
	printWarningsTo(os.Stderr, warnings)
}

func printWarningsTo(output io.Writer, warnings []compiler.Diagnostic) {
	for _, warning := range warnings {
		fmt.Fprintln(output, formatDiagnosticWithSnippet(warning))
	}
}

func parseCLI(args []string) (cliConfig, error) {
	cfg := cliConfig{
		mode:          "compile",
		targetName:    "host",
		warningPolicy: "default",
	}
	if len(args) > 0 && (args[0] == "compile" || args[0] == "run" || args[0] == "init" || args[0] == "test") {
		cfg.mode = args[0]
		args = args[1:]
	}
	if cfg.mode == "init" {
		if len(args) > 1 {
			return cliConfig{}, fmt.Errorf("usage: %s", usageForMode(cfg.mode))
		}
		if len(args) == 1 {
			cfg.initPath = args[0]
		}
		return cfg, nil
	}

	flags := flag.NewFlagSet("jayess", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&cfg.emit, "emit", "", "output kind: llvm, obj, or exe")
	flags.StringVar(&cfg.optimizationLevel, "opt", "", "optimization level: O0, O1, O2, O3, or Oz")
	flags.StringVar(&cfg.targetCPU, "cpu", "", "target CPU name passed to clang")
	flags.Var(&cfg.targetFeatures, "feature", "target feature passed to clang; repeatable or comma-separated, such as +sse2 or -avx")
	flags.StringVar(&cfg.relocationModel, "reloc", "", "relocation model: pic, pie, or static")
	flags.StringVar(&cfg.codeModel, "code-model", "", "code model: small, medium, large, or kernel")
	flags.StringVar(&cfg.targetName, "target", "host", "target name such as windows-x64 or darwin-arm64")
	flags.StringVar(&cfg.output, "o", "", "output file path")
	flags.StringVar(&cfg.warningPolicy, "warnings", "default", "warning policy: default, none, or error")
	flags.Var(&cfg.allowedWarningCategories, "allow-warning", "warning category to allow when --warnings=error; repeatable or comma-separated")
	if err := flags.Parse(args); err != nil {
		return cliConfig{}, fmt.Errorf("usage: %s", usageForMode(cfg.mode))
	}
	remaining := flags.Args()
	if cfg.mode == "test" && len(remaining) == 0 {
		cfg.inputPath = "."
		return cfg, nil
	}
	if len(remaining) == 0 {
		return cliConfig{}, fmt.Errorf("usage: %s", usageForMode(cfg.mode))
	}
	cfg.inputPath = remaining[0]
	if cfg.mode == "run" {
		cfg.programArgs = remaining[1:]
	} else if len(remaining) != 1 {
		return cliConfig{}, fmt.Errorf("usage: %s", usageForMode(cfg.mode))
	}
	return cfg, nil
}

func usageForMode(mode string) string {
	switch mode {
	case "init":
		return "jayess init [directory]"
	case "run":
		return "jayess run [--target=<name>] [--opt=O0|O1|O2|O3|Oz] [--cpu=<name>] [--feature=<flag>] [--reloc=pic|pie|static] [--code-model=small|medium|large|kernel] [--warnings=default|none|error] [--allow-warning=<category>] [-o output] <input.js> [args...]"
	case "test":
		return "jayess test [--target=<name>] [--opt=O0|O1|O2|O3|Oz] [--cpu=<name>] [--feature=<flag>] [--reloc=pic|pie|static] [--code-model=small|medium|large|kernel] [--warnings=default|none|error] [--allow-warning=<category>] [path|file.test.js]"
	default:
		return "jayess [--target=<name>] [--emit=llvm|bc|obj|lib|shared|exe] [--opt=O0|O1|O2|O3|Oz] [--cpu=<name>] [--feature=<flag>] [--reloc=pic|pie|static] [--code-model=small|medium|large|kernel] [--warnings=default|none|error] [--allow-warning=<category>] [-o output] <input.js>"
	}
}

func defaultOutputPath(inputPath, emit string) string {
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	switch emit {
	case "exe":
		if runtime.GOOS == "windows" {
			return filepath.Join("build", base+".exe")
		}
		return filepath.Join("build", base)
	case "obj":
		return filepath.Join("build", base+".o")
	case "lib":
		return filepath.Join("build", "lib"+base+".a")
	case "shared":
		switch runtime.GOOS {
		case "windows":
			return filepath.Join("build", base+".dll")
		case "darwin":
			return filepath.Join("build", "lib"+base+".dylib")
		default:
			return filepath.Join("build", "lib"+base+".so")
		}
	case "bc":
		return filepath.Join("build", base+".bc")
	default:
		return filepath.Join("build", base+".ll")
	}
}

func defaultEmitMode() string {
	if runtime.GOOS == "windows" {
		return "exe"
	}
	return "llvm"
}

func isSupportedOptimizationLevel(level string) bool {
	switch level {
	case "O0", "O1", "O2", "O3", "Oz":
		return true
	default:
		return false
	}
}

func isSupportedRelocationModel(model string) bool {
	switch model {
	case "pic", "pie", "static":
		return true
	default:
		return false
	}
}

func isSupportedCodeModel(model string) bool {
	switch model {
	case "small", "medium", "large", "kernel":
		return true
	default:
		return false
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func exitDiagnostic(inputPath string, err error) {
	fmt.Fprintln(os.Stderr, formatCompileErrorWithSnippet(inputPath, err))
	os.Exit(1)
}

func formatDiagnostic(d compiler.Diagnostic) string {
	location := ""
	if d.File != "" {
		location = d.File
		if d.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, d.Line)
			if d.Column > 0 {
				location = fmt.Sprintf("%s:%d", location, d.Column)
			}
		}
		location += ": "
	}
	severity := d.Severity
	if severity == "" {
		severity = "warning"
	}
	label := severity
	if d.Code != "" {
		label = fmt.Sprintf("%s[%s]", label, d.Code)
	}
	if d.Category != "" {
		label = fmt.Sprintf("%s/%s", label, d.Category)
	}
	return fmt.Sprintf("%s%s: %s", location, label, d.Message)
}

func formatDiagnosticWithSnippet(d compiler.Diagnostic) string {
	base := formatDiagnostic(d)
	snippet := readSourceLine(d.File, d.Line)
	if snippet != "" && d.Column > 0 {
		base = fmt.Sprintf("%s\n%s\n%s^", base, snippet, strings.Repeat(" ", max(d.Column-1, 0)))
	}
	for _, note := range d.Notes {
		base = fmt.Sprintf("%s\nnote: %s", base, note)
	}
	return base
}

func formatCompileErrorWithSnippet(inputPath string, err error) string {
	if err == nil {
		return ""
	}
	var compileErr *compiler.CompileError
	if errors.As(err, &compileErr) {
		diagnostic := compileErr.Diagnostic
		if diagnostic.File == "" {
			diagnostic.File = inputPath
		}
		return formatDiagnosticWithSnippet(diagnostic)
	}
	return err.Error()
}

func readSourceLine(path string, line int) string {
	if path == "" || line <= 0 {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if line-1 < 0 || line-1 >= len(lines) {
		return ""
	}
	return lines[line-1]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
