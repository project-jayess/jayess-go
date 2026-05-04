package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
)

func TestParserRejectsJSXElementWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const view = <Button />;`)
	if err == nil {
		t.Fatalf("expected unsupported JSX element error")
	}
	if !strings.Contains(err.Error(), "JSX and angle-bracket type assertions are not supported") {
		t.Fatalf("expected clear JSX diagnostic, got %v", err)
	}
}

func TestParserRejectsAngleBracketTypeAssertionWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const value = <Widget>item;`)
	if err == nil {
		t.Fatalf("expected unsupported angle-bracket type assertion error")
	}
	if !strings.Contains(err.Error(), "JSX and angle-bracket type assertions are not supported") {
		t.Fatalf("expected clear angle-bracket diagnostic, got %v", err)
	}
}

func TestParserStillParsesLessThanComparison(t *testing.T) {
	expr := parseExpression(t, `left < right`)
	comparison := requireType[*ast.ComparisonExpression](t, expr)
	if comparison.Operator != ast.OperatorLt {
		t.Fatalf("expected less-than operator, got %q", comparison.Operator)
	}
}
