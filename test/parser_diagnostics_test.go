package test

import (
	"strings"
	"testing"

	"jayess-go/parser"
)

func TestParserReportsUnterminatedStringDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const value = "missing`)
	requireParserError(t, err, "unterminated string")
}

func TestParserReportsUnterminatedTemplateDiagnostic(t *testing.T) {
	_, err := parseProgramError("const value = `missing ${name}")
	requireParserError(t, err, "unterminated template")
}

func TestParserReportsUnexpectedCharacterDiagnostic(t *testing.T) {
	_, err := parseProgramError("\\")
	requireParserError(t, err, "1:1: unexpected character '\\\\'")
}

func TestParserReportsExpectedTokenBeforeEOF(t *testing.T) {
	_, err := parseProgramError("if (ready")
	requireParserError(t, err, "expected ) before end of file")
}

func TestParserDiagnosticIncludesSourceSpan(t *testing.T) {
	_, err := parseProgramError("const value = 1\nconst other;")
	diagnostic, ok := err.(*parser.DiagnosticError)
	if !ok {
		t.Fatalf("expected parser diagnostic error, got %T", err)
	}
	if diagnostic.Line != 2 || diagnostic.Column != 1 {
		t.Fatalf("expected diagnostic at 2:1, got %d:%d", diagnostic.Line, diagnostic.Column)
	}
	if diagnostic.Message != "const declaration requires an initializer" {
		t.Fatalf("unexpected diagnostic message %q", diagnostic.Message)
	}
}

func TestParserDiagnosticStringIncludesSourceSpan(t *testing.T) {
	_, err := parseProgramError("var first = 1 var second = 2")
	requireParserError(t, err, "1:15:")
	requireParserError(t, err, "expected statement terminator")
}

func requireParserError(t *testing.T, err error, message string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected parser error containing %q", message)
	}
	if !strings.Contains(err.Error(), message) {
		t.Fatalf("expected parser error containing %q, got %v", message, err)
	}
}
