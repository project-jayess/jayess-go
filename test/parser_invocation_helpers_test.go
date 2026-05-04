package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserBindInvocationHelper(t *testing.T) {
	expr := parseExpression(t, `identity.bind(null, 1)`)
	call := requireType[*ast.InvokeExpression](t, expr)
	member := requireType[*ast.MemberExpression](t, call.Callee)
	if member.Property != "bind" {
		t.Fatalf("expected bind helper, got %q", member.Property)
	}
	requireType[*ast.Identifier](t, member.Target)
}

func TestParserCallInvocationHelper(t *testing.T) {
	expr := parseExpression(t, `identity.call(null, 1)`)
	call := requireType[*ast.InvokeExpression](t, expr)
	member := requireType[*ast.MemberExpression](t, call.Callee)
	if member.Property != "call" {
		t.Fatalf("expected call helper, got %q", member.Property)
	}
}

func TestParserApplyInvocationHelper(t *testing.T) {
	expr := parseExpression(t, `identity.apply(null, args)`)
	call := requireType[*ast.InvokeExpression](t, expr)
	member := requireType[*ast.MemberExpression](t, call.Callee)
	if member.Property != "apply" {
		t.Fatalf("expected apply helper, got %q", member.Property)
	}
}
