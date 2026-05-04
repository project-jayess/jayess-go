package test

import (
	"strings"
	"testing"

	"jayess-go/diagnostics"
)

func TestDiagnosticsTypeErrorsHaveSourceLocations(t *testing.T) {
	diagnostic := diagnostics.TypeError(
		diagnostics.SourceLocation{File: "main.js", Line: 3, Column: 9},
		diagnostics.TypeMismatch,
		"cannot assign string to number",
	)

	if diagnostic.Severity != diagnostics.ErrorSeverity {
		t.Fatalf("expected error severity, got %s", diagnostic.Severity)
	}
	if !strings.Contains(diagnostic.String(), "main.js:3:9") {
		t.Fatalf("expected source location in diagnostic string, got %q", diagnostic.String())
	}
	if !hasTypeErrorKind(diagnostics.TypeErrorKinds(), diagnostics.InvalidCallTarget) {
		t.Fatalf("expected invalid call target type error kind")
	}
}

func hasTypeErrorKind(values []diagnostics.TypeErrorKind, want diagnostics.TypeErrorKind) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
