package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
