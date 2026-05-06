package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
)

func TestParserRejectsLetDeclarationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`let value = 1;`)
	if err == nil {
		t.Fatalf("expected unsupported let declaration error")
	}
	if !strings.Contains(err.Error(), "let declarations are not supported") {
		t.Fatalf("expected clear let diagnostic, got %v", err)
	}
}

func TestParserStillAllowsLetPropertyName(t *testing.T) {
	expr := parseExpression(t, `item.let`)
	member := requireType[*ast.MemberExpression](t, expr)
	if member.Property != "let" {
		t.Fatalf("expected let property, got %q", member.Property)
	}
}

func TestParserRejectsPublicModifierWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`public function main() {}`)
	if err == nil {
		t.Fatalf("expected unsupported public modifier error")
	}
	if !strings.Contains(err.Error(), "public is not supported") {
		t.Fatalf("expected clear public diagnostic, got %v", err)
	}
}

func TestParserRejectsTopLevelPrivateWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`private const value = 1;`)
	if err == nil {
		t.Fatalf("expected unsupported top-level private error")
	}
	if !strings.Contains(err.Error(), "top-level private is not supported") {
		t.Fatalf("expected clear private diagnostic, got %v", err)
	}
}

func TestParserStillAllowsVisibilityKeywordPropertyNames(t *testing.T) {
	expr := parseExpression(t, `item.public + item.private`)
	binary := requireType[*ast.BinaryExpression](t, expr)
	left := requireType[*ast.MemberExpression](t, binary.Left)
	right := requireType[*ast.MemberExpression](t, binary.Right)
	if left.Property != "public" || right.Property != "private" {
		t.Fatalf("unexpected visibility keyword properties: %q %q", left.Property, right.Property)
	}
}

func TestParserRejectsWithStatementWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`with (item) { value = name; }`)
	if err == nil {
		t.Fatalf("expected unsupported with statement error")
	}
	if !strings.Contains(err.Error(), "with statements are not supported") {
		t.Fatalf("expected clear with diagnostic, got %v", err)
	}
}

func TestParserStillAllowsWithPropertyName(t *testing.T) {
	expr := parseExpression(t, `item.with`)
	member := requireType[*ast.MemberExpression](t, expr)
	if member.Property != "with" {
		t.Fatalf("expected with property, got %q", member.Property)
	}
}

func TestParserRejectsEnumDeclarationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`enum Color { Red, Blue }`)
	if err == nil {
		t.Fatalf("expected unsupported enum declaration error")
	}
	if !strings.Contains(err.Error(), "enum declarations are not supported") {
		t.Fatalf("expected clear enum diagnostic, got %v", err)
	}
}

func TestParserStillAllowsEnumPropertyName(t *testing.T) {
	expr := parseExpression(t, `item.enum`)
	member := requireType[*ast.MemberExpression](t, expr)
	if member.Property != "enum" {
		t.Fatalf("expected enum property, got %q", member.Property)
	}
}

func TestParserRejectsTypeAliasWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`type User = { name: string };`)
	if err == nil {
		t.Fatalf("expected unsupported type alias error")
	}
	if !strings.Contains(err.Error(), "type aliases are not supported") {
		t.Fatalf("expected clear type alias diagnostic, got %v", err)
	}
}

func TestParserRejectsInterfaceDeclarationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`interface User { name: string }`)
	if err == nil {
		t.Fatalf("expected unsupported interface declaration error")
	}
	if !strings.Contains(err.Error(), "interface declarations are not supported") {
		t.Fatalf("expected clear interface diagnostic, got %v", err)
	}
}

func TestParserRejectsAmbientDeclarationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`declare const value: number;`)
	if err == nil {
		t.Fatalf("expected unsupported ambient declaration error")
	}
	if !strings.Contains(err.Error(), "ambient declarations are not supported") {
		t.Fatalf("expected clear ambient declaration diagnostic, got %v", err)
	}
}

func TestParserRejectsNamespaceDeclarationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`namespace App { export const value = 1; }`)
	if err == nil {
		t.Fatalf("expected unsupported namespace declaration error")
	}
	if !strings.Contains(err.Error(), "namespace declarations are not supported") {
		t.Fatalf("expected clear namespace declaration diagnostic, got %v", err)
	}
}

func TestParserStillAllowsDeclareAndNamespacePropertyNames(t *testing.T) {
	expr := parseExpression(t, `item.declare + item.namespace`)
	binary := requireType[*ast.BinaryExpression](t, expr)
	left := requireType[*ast.MemberExpression](t, binary.Left)
	right := requireType[*ast.MemberExpression](t, binary.Right)
	if left.Property != "declare" || right.Property != "namespace" {
		t.Fatalf("unexpected declaration keyword-like properties: %q %q", left.Property, right.Property)
	}
}

func TestParserStillAllowsTypeAndInterfacePropertyNames(t *testing.T) {
	expr := parseExpression(t, `item.type + item.interface`)
	binary := requireType[*ast.BinaryExpression](t, expr)
	left := requireType[*ast.MemberExpression](t, binary.Left)
	right := requireType[*ast.MemberExpression](t, binary.Right)
	if left.Property != "type" || right.Property != "interface" {
		t.Fatalf("unexpected type keyword-like properties: %q %q", left.Property, right.Property)
	}
}
