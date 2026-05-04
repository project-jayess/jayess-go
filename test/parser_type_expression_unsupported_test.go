package test

import (
	"strings"
	"testing"
)

func TestParserRejectsAsTypeAssertionWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const value = item as Widget;`)
	if err == nil {
		t.Fatalf("expected unsupported type assertion error")
	}
	if !strings.Contains(err.Error(), "type assertions are not supported") {
		t.Fatalf("expected clear type assertion diagnostic, got %v", err)
	}
}

func TestParserRejectsAsTypeAssertionInsideCallWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`use(item as Widget);`)
	if err == nil {
		t.Fatalf("expected unsupported type assertion error")
	}
	if !strings.Contains(err.Error(), "type assertions are not supported") {
		t.Fatalf("expected clear type assertion diagnostic, got %v", err)
	}
}

func TestParserRejectsSatisfiesExpressionWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const value = item satisfies Widget;`)
	if err == nil {
		t.Fatalf("expected unsupported satisfies expression error")
	}
	if !strings.Contains(err.Error(), "satisfies expressions are not supported") {
		t.Fatalf("expected clear satisfies diagnostic, got %v", err)
	}
}

func TestParserRejectsSatisfiesExpressionInsideGroupWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const value = (item satisfies Widget);`)
	if err == nil {
		t.Fatalf("expected unsupported satisfies expression error")
	}
	if !strings.Contains(err.Error(), "satisfies expressions are not supported") {
		t.Fatalf("expected clear satisfies diagnostic, got %v", err)
	}
}

func TestParserRejectsNonNullAssertionWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const value = item!;`)
	if err == nil {
		t.Fatalf("expected unsupported non-null assertion error")
	}
	if !strings.Contains(err.Error(), "non-null assertions are not supported") {
		t.Fatalf("expected clear non-null assertion diagnostic, got %v", err)
	}
}

func TestParserRejectsNonNullAssertionInsideArrayWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const values = [item!];`)
	if err == nil {
		t.Fatalf("expected unsupported non-null assertion error")
	}
	if !strings.Contains(err.Error(), "non-null assertions are not supported") {
		t.Fatalf("expected clear non-null assertion diagnostic, got %v", err)
	}
}

func TestParserStillAllowsAsAndSatisfiesAcrossStatementBoundary(t *testing.T) {
	program := parseProgram(t, `
		value
		as;
		satisfies;
	`)
	if len(program.Statements) != 3 {
		t.Fatalf("expected three expression statements, got %d", len(program.Statements))
	}
}
