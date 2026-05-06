package test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"jayess-go/diagnostics"
	"jayess-go/lexer"
	"jayess-go/lowering"
	"jayess-go/parser"
	"jayess-go/resolver"
	"jayess-go/semantic"
	"jayess-go/tooling"
)

func TestCompilerPerformanceBaselineForLargeSources(t *testing.T) {
	root := cliRepoRoot(t)
	workDir := cliTempDir(t, root, "performance-baseline-*")
	sourcePath := filepath.Join(workDir, "large.js")
	source := largeCompilerLikeSource(512)
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	var programParsed bool
	phases := map[string]tooling.PhaseFunc{
		"parse": func() error {
			program, err := parser.New(lexer.New(source)).ParseProgram()
			programParsed = program != nil
			return err
		},
		"semantic": func() error {
			program, err := parser.New(lexer.New(source)).ParseProgram()
			if err != nil {
				return err
			}
			return semantic.New().Analyze(program)
		},
		"lowering": func() error {
			program, err := parser.New(lexer.New(source)).ParseProgram()
			if err != nil {
				return err
			}
			lowering.MainReturnCode(program)
			return nil
		},
		"backend": func() error {
			runJayessCLI(t, root, "compile", "--target=linux-x64", "--emit=llvm", "-o", filepath.Join(workDir, "large.ll"), sourcePath)
			return nil
		},
	}
	measurements, err := tooling.MeasurePhases(phases, []string{"parse", "semantic", "lowering", "backend"})
	if err != nil {
		t.Fatal(err)
	}
	if !programParsed || len(measurements) != 4 {
		t.Fatalf("expected four phase measurements, got %#v", measurements)
	}
	writePerformanceBaseline(t, filepath.Join(workDir, "baseline.txt"), measurements)
}

func TestCompilerPerformanceBaselineForModuleAndDiagnosticScale(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 128; i++ {
		next := ""
		if i+1 < 128 {
			next = fmt.Sprintf("import \"./module_%03d.js\";\n", i+1)
		}
		writeFile(t, filepath.Join(root, fmt.Sprintf("module_%03d.js", i)), next+"export const value = 1;\n")
	}
	project, err := resolver.LoadProject(filepath.Join(root, "module_000.js"))
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 || len(project.Modules) != 128 {
		t.Fatalf("expected bounded module graph, got modules=%d diagnostics=%#v", len(project.Modules), project.Diagnostics)
	}

	var collection diagnostics.Collection
	for i := 127; i >= 0; i-- {
		collection.AddError("JY-TEST", diagnostics.SourceLocation{File: "module.js", Line: i + 1, Column: 1}, "diagnostic")
	}
	ordered := collection.Diagnostics()
	if len(ordered) != 128 || ordered[0].Span.Start.Line != 1 || ordered[127].Span.Start.Line != 128 {
		t.Fatalf("expected deterministic diagnostic ordering, got %#v", ordered)
	}
}

func largeCompilerLikeSource(count int) string {
	source := "function main(args) {\nvar total = 0;\n"
	for i := 0; i < count; i++ {
		source += fmt.Sprintf("total = total + %d;\n", i%7)
	}
	return source + "return total === -1 ? 1 : 0;\n}\n"
}

func writePerformanceBaseline(t *testing.T, path string, measurements []tooling.PhaseMeasurement) {
	t.Helper()
	content := ""
	for _, measurement := range measurements {
		if measurement.Duration < 0*time.Nanosecond {
			t.Fatalf("invalid duration for %s", measurement.Name)
		}
		content += fmt.Sprintf("%s=%s\n", measurement.Name, measurement.Duration)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	requireFile(t, path)
}
