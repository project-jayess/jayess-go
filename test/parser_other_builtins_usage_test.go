package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserDateConstructorAndMethodCall(t *testing.T) {
	expr := parseExpression(t, `today.getTime()`)
	call := requireType[*ast.InvokeExpression](t, expr)
	member := requireType[*ast.MemberExpression](t, call.Callee)
	if member.Property != "getTime" {
		t.Fatalf("expected getTime method, got %q", member.Property)
	}
}

func TestParserRegExpConstructorAndMethodCall(t *testing.T) {
	expr := parseExpression(t, `pattern.test(value)`)
	call := requireType[*ast.InvokeExpression](t, expr)
	member := requireType[*ast.MemberExpression](t, call.Callee)
	if member.Property != "test" {
		t.Fatalf("expected test method, got %q", member.Property)
	}
}

func TestParserTypedArrayConstructor(t *testing.T) {
	expr := parseExpression(t, `new Uint8Array(buffer)`)
	newExpr := requireType[*ast.NewExpression](t, expr)
	requireType[*ast.Identifier](t, newExpr.Callee)
}

func TestParserTypedArrayIndexing(t *testing.T) {
	expr := parseExpression(t, `bytes[0]`)
	index := requireType[*ast.IndexExpression](t, expr)
	requireType[*ast.Identifier](t, index.Target)
	requireType[*ast.NumberLiteral](t, index.Index)
}

func TestParserObjectConstructorAndStaticMethodCall(t *testing.T) {
	expr := parseExpression(t, `Object.create(proto)`)
	call := requireType[*ast.InvokeExpression](t, expr)
	member := requireType[*ast.MemberExpression](t, call.Callee)
	if member.Property != "create" {
		t.Fatalf("expected create method, got %q", member.Property)
	}
	requireType[*ast.Identifier](t, member.Target)
}
