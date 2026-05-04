package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserMapConstructor(t *testing.T) {
	expr := parseExpression(t, `new Map()`)
	newExpr := requireType[*ast.NewExpression](t, expr)
	requireType[*ast.Identifier](t, newExpr.Callee)
}

func TestParserMapMethodCall(t *testing.T) {
	expr := parseExpression(t, `items.set("name", value)`)
	call := requireType[*ast.InvokeExpression](t, expr)
	if len(call.Arguments) != 2 {
		t.Fatalf("expected two set arguments, got %d", len(call.Arguments))
	}
	member := requireType[*ast.MemberExpression](t, call.Callee)
	if member.Property != "set" {
		t.Fatalf("expected set method, got %q", member.Property)
	}
	requireType[*ast.Identifier](t, member.Target)
}

func TestParserSetConstructor(t *testing.T) {
	expr := parseExpression(t, `new Set()`)
	newExpr := requireType[*ast.NewExpression](t, expr)
	requireType[*ast.Identifier](t, newExpr.Callee)
}

func TestParserSetMethodCall(t *testing.T) {
	expr := parseExpression(t, `items.add(value)`)
	call := requireType[*ast.InvokeExpression](t, expr)
	member := requireType[*ast.MemberExpression](t, call.Callee)
	if member.Property != "add" {
		t.Fatalf("expected add method, got %q", member.Property)
	}
	requireType[*ast.Identifier](t, member.Target)
}
