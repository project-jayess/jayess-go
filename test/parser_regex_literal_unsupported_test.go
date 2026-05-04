package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
)

func TestParserRejectsRegexLiteralWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const matcher = /a+/;`)
	if err == nil {
		t.Fatalf("expected unsupported regular expression literal error")
	}
	if !strings.Contains(err.Error(), "regular expression literals are not supported") {
		t.Fatalf("expected clear regex literal diagnostic, got %v", err)
	}
}

func TestParserStillParsesDivisionExpression(t *testing.T) {
	expr := parseExpression(t, `total / count`)
	binary := requireType[*ast.BinaryExpression](t, expr)
	if binary.Operator != ast.OperatorDiv {
		t.Fatalf("expected divide operator, got %q", binary.Operator)
	}
}
