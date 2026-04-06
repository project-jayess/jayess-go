package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"jayess-go/backend"
	"jayess-go/compiler"
	"jayess-go/target"
)

func main() {
	var emit string
	var targetName string
	var output string

	flag.StringVar(&emit, "emit", "", "output kind: llvm or exe")
	flag.StringVar(&targetName, "target", "host", "target name such as windows-x64 or darwin-arm64")
	flag.StringVar(&output, "o", "", "output file path")
	flag.Parse()

	if flag.NArg() != 1 {
		exitf("usage: jayess [--target=<name>] [--emit=llvm|exe] [-o output] <input.js>")
	}

	inputPath := flag.Arg(0)

	targetTriple, err := target.FromName(targetName)
	if err != nil {
		exitf("resolve target: %v", err)
	}

	if emit == "" {
		emit = defaultEmitMode()
	}

	if output == "" {
		output = defaultOutputPath(inputPath, emit)
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		exitf("create output directory: %v", err)
	}

	opts := compiler.Options{TargetTriple: targetTriple}
	result, err := compiler.CompilePath(inputPath, opts)
	if err != nil {
		exitDiagnostic(inputPath, err)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintln(os.Stderr, formatDiagnosticWithSnippet(warning))
	}

	switch emit {
	case "llvm":
		if err := os.WriteFile(output, result.LLVMIR, 0o644); err != nil {
			exitf("write output: %v", err)
		}
	case "exe":
		tc, err := backend.DetectToolchain()
		if err != nil {
			exitf("detect LLVM toolchain: %v", err)
		}
		if err := tc.BuildExecutable(result, opts, output); err != nil {
			exitf("build executable: %v", err)
		}
	default:
		exitf("unsupported emit mode %q", emit)
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
