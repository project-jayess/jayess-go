package test

import (
	"testing"

	"jayess-go/escape"
)

func TestEscapeLifetimeDiagnosticReportsEscapingBinding(t *testing.T) {
	program := parseProgram(t, "function make() {\nconst value = {};\nreturn value;\n}")

	diagnostic := findLifetimeDiagnostic(t, escape.LifetimeDiagnostics(program), "value")
	if !diagnostic.Escaping {
		t.Fatalf("expected value diagnostic to be escaping")
	}
	if diagnostic.Line != 2 || diagnostic.Column != 1 {
		t.Fatalf("expected diagnostic at 2:1, got %d:%d", diagnostic.Line, diagnostic.Column)
	}
	if diagnostic.Message != "value escapes lexical scope; skip scope-exit cleanup" {
		t.Fatalf("unexpected diagnostic message %q", diagnostic.Message)
	}
}

func TestEscapeLifetimeDiagnosticReportsCleanupEligibleBinding(t *testing.T) {
	program := parseProgram(t, "function make() {\nconst value = {};\nreturn 1;\n}")

	diagnostic := findLifetimeDiagnostic(t, escape.LifetimeDiagnostics(program), "value")
	if diagnostic.Escaping {
		t.Fatalf("expected value diagnostic to be non-escaping")
	}
	if diagnostic.Message != "value does not escape lexical scope; eligible for scope-exit cleanup" {
		t.Fatalf("unexpected diagnostic message %q", diagnostic.Message)
	}
}

func findLifetimeDiagnostic(t *testing.T, diagnostics []escape.Diagnostic, binding string) escape.Diagnostic {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Binding == binding {
			return diagnostic
		}
	}
	t.Fatalf("expected lifetime diagnostic for %s", binding)
	return escape.Diagnostic{}
}
