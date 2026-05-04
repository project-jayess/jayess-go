package test

import (
	"strings"
	"testing"

	"jayess-go/diagnostics"
)

func TestDiagnosticsRuntimeExceptionStackTraceIncludesSourceLocations(t *testing.T) {
	exception := diagnostics.NewException("TypeError", "value is not callable", []diagnostics.StackFrame{
		{
			Function: "run",
			Module:   "main.js",
			Location: diagnostics.SourceLocation{File: "main.js", Line: 8, Column: 13},
		},
		{
			Function: "start",
			Module:   "bootstrap.js",
			Location: diagnostics.SourceLocation{File: "bootstrap.js", Line: 2, Column: 1},
		},
	})

	formatted := exception.Stack.Format()
	if !strings.Contains(formatted, "run") || !strings.Contains(formatted, "main.js:8:13") {
		t.Fatalf("expected formatted stack trace with source location, got %q", formatted)
	}
	location, ok := exception.Stack.TopLocation()
	if !ok || location.File != "main.js" || location.Line != 8 {
		t.Fatalf("unexpected top stack location: %#v", location)
	}
}

func TestDiagnosticsUncaughtExceptionDiagnostic(t *testing.T) {
	exception := diagnostics.NewException("Error", "boom", []diagnostics.StackFrame{
		{Function: "main", Location: diagnostics.SourceLocation{File: "app.js", Line: 1, Column: 5}},
	})

	diagnostic := exception.UncaughtDiagnostic()
	if diagnostic.Code != "JY-RUNTIME-uncaught-exception" {
		t.Fatalf("unexpected uncaught diagnostic code %q", diagnostic.Code)
	}
	if !strings.Contains(diagnostic.Message, "uncaught Error: boom") {
		t.Fatalf("unexpected uncaught diagnostic message %q", diagnostic.Message)
	}
	if diagnostic.Location.File != "app.js" || diagnostic.Location.Column != 5 {
		t.Fatalf("unexpected uncaught diagnostic source location: %#v", diagnostic.Location)
	}
	if !hasRuntimeErrorKind(diagnostics.RuntimeErrorKinds(), diagnostics.UncaughtException) {
		t.Fatal("expected uncaught exception runtime error kind")
	}
}

func hasRuntimeErrorKind(values []diagnostics.RuntimeErrorKind, want diagnostics.RuntimeErrorKind) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
