package test

import (
	"testing"

	"jayess-go/semantic"
)

func TestSemanticDiagnosticIncludesSourceSpan(t *testing.T) {
	err := analyzeSource(t, "missing;\n")
	diagnostic, ok := err.(*semantic.DiagnosticError)
	if !ok {
		t.Fatalf("expected semantic diagnostic error, got %T", err)
	}
	if diagnostic.Line != 1 || diagnostic.Column != 1 {
		t.Fatalf("expected diagnostic at 1:1, got %d:%d", diagnostic.Line, diagnostic.Column)
	}
	if diagnostic.Message != "use of missing before declaration" {
		t.Fatalf("unexpected diagnostic message %q", diagnostic.Message)
	}
}

func TestSemanticDiagnosticStringIncludesSourceSpan(t *testing.T) {
	err := analyzeSource(t, "const value = missing;\n")
	requireSemanticError(t, err, "1:15: use of missing before declaration")
}

func TestSemanticModuleDiagnosticIncludesSourceSpan(t *testing.T) {
	err := analyzeSource(t, `import { add } from "/math.js";`)
	diagnostic, ok := err.(*semantic.DiagnosticError)
	if !ok {
		t.Fatalf("expected semantic diagnostic error, got %T", err)
	}
	if diagnostic.Line != 1 || diagnostic.Column != 1 {
		t.Fatalf("expected diagnostic at 1:1, got %d:%d", diagnostic.Line, diagnostic.Column)
	}
	if diagnostic.Message != `unsupported module source "/math.js"; expected relative or package specifier` {
		t.Fatalf("unexpected diagnostic message %q", diagnostic.Message)
	}
}

func TestSemanticConstAssignmentDiagnosticIncludesMutationHint(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		value = 2;
	`)
	diagnostic, ok := err.(*semantic.DiagnosticError)
	if !ok {
		t.Fatalf("expected semantic diagnostic error, got %T", err)
	}
	if diagnostic.Message != "assignment to const value; use var for mutable bindings" {
		t.Fatalf("unexpected diagnostic message %q", diagnostic.Message)
	}
}
