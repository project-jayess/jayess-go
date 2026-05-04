package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserStringConcatenation(t *testing.T) {
	expr := parseExpression(t, `"hello, " + name + "!"`)
	secondAdd := requireType[*ast.BinaryExpression](t, expr)
	if secondAdd.Operator != ast.OperatorAdd {
		t.Fatalf("expected string concatenation at root, got %q", secondAdd.Operator)
	}
	firstAdd := requireType[*ast.BinaryExpression](t, secondAdd.Left)
	if firstAdd.Operator != ast.OperatorAdd {
		t.Fatalf("expected nested string concatenation, got %q", firstAdd.Operator)
	}
	requireType[*ast.StringLiteral](t, firstAdd.Left)
	requireType[*ast.Identifier](t, firstAdd.Right)
	requireType[*ast.StringLiteral](t, secondAdd.Right)
}

func TestParserStringIndexing(t *testing.T) {
	expr := parseExpression(t, `"hello"[0]`)
	index := requireType[*ast.IndexExpression](t, expr)
	literal := requireType[*ast.StringLiteral](t, index.Target)
	if literal.Value != "hello" {
		t.Fatalf("expected hello literal, got %q", literal.Value)
	}
	requireType[*ast.NumberLiteral](t, index.Index)
}

func TestParserStringLengthMember(t *testing.T) {
	expr := parseExpression(t, `"hello".length`)
	member := requireType[*ast.MemberExpression](t, expr)
	requireType[*ast.StringLiteral](t, member.Target)
	if member.Property != "length" {
		t.Fatalf("expected length member, got %q", member.Property)
	}
}

func TestParserTemplateStringInterpolation(t *testing.T) {
	expr := parseExpression(t, "`hello ${name}`")
	template := requireType[*ast.TemplateLiteral](t, expr)
	if template.Value != "hello ${name}" {
		t.Fatalf("unexpected template literal value %q", template.Value)
	}
	if len(template.Expressions) != 1 {
		t.Fatalf("expected one template expression, got %d", len(template.Expressions))
	}
	requireType[*ast.Identifier](t, template.Expressions[0])
}

func TestParserUnicodeStringLiteral(t *testing.T) {
	expr := parseExpression(t, `"こんにちは"`)
	literal := requireType[*ast.StringLiteral](t, expr)
	if literal.Value != "こんにちは" {
		t.Fatalf("expected unicode literal, got %q", literal.Value)
	}
}

func TestParserUnicodeIdentifierInStringConcatenation(t *testing.T) {
	expr := parseExpression(t, `"π=" + π`)
	add := requireType[*ast.BinaryExpression](t, expr)
	requireType[*ast.StringLiteral](t, add.Left)
	identifier := requireType[*ast.Identifier](t, add.Right)
	if identifier.Name != "π" {
		t.Fatalf("expected unicode identifier π, got %q", identifier.Name)
	}
}
