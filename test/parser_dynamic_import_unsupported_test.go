package test

import (
	"strings"
	"testing"
)

func TestParserRejectsDynamicImportExpressionWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const mod = import("./mod.js");`)
	if err == nil {
		t.Fatalf("expected unsupported dynamic import expression error")
	}
	if !strings.Contains(err.Error(), "dynamic import expressions are not supported") {
		t.Fatalf("expected clear dynamic import diagnostic, got %v", err)
	}
}

func TestParserRejectsDynamicImportStatementWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`import("./setup.js");`)
	if err == nil {
		t.Fatalf("expected unsupported dynamic import statement error")
	}
	if !strings.Contains(err.Error(), "dynamic import expressions are not supported") {
		t.Fatalf("expected clear dynamic import diagnostic, got %v", err)
	}
}

func TestParserStillAllowsImportMetaExpression(t *testing.T) {
	program := parseProgram(t, `const url = import.meta.url;`)
	if len(program.Statements) != 1 {
		t.Fatalf("expected one statement, got %d", len(program.Statements))
	}
}
