package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
)

func TestParserRejectsUnsupportedAbstractClassDeclaration(t *testing.T) {
	_, err := parseProgramError(`abstract class Widget {}`)
	if err == nil {
		t.Fatalf("expected abstract class error")
	}
	if !strings.Contains(err.Error(), "abstract modifiers are not supported") {
		t.Fatalf("expected unsupported abstract diagnostic, got %v", err)
	}
}

func TestParserRejectsUnsupportedExportAbstractClassDeclaration(t *testing.T) {
	_, err := parseProgramError(`export abstract class Widget {}`)
	if err == nil {
		t.Fatalf("expected export abstract class error")
	}
	if !strings.Contains(err.Error(), "abstract modifiers are not supported") {
		t.Fatalf("expected unsupported abstract diagnostic, got %v", err)
	}
}

func TestParserRejectsUnsupportedDefaultAbstractClassDeclaration(t *testing.T) {
	_, err := parseProgramError(`export default abstract class Widget {}`)
	if err == nil {
		t.Fatalf("expected default abstract class error")
	}
	if !strings.Contains(err.Error(), "abstract modifiers are not supported") {
		t.Fatalf("expected unsupported abstract diagnostic, got %v", err)
	}
}

func TestParserStillAllowsAbstractPropertyName(t *testing.T) {
	expr := parseExpression(t, `item.abstract`)
	member := requireType[*ast.MemberExpression](t, expr)
	if member.Property != "abstract" {
		t.Fatalf("expected abstract property, got %q", member.Property)
	}
}
