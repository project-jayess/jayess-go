package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserPromiseErrorPropagationChain(t *testing.T) {
	program := parseProgram(t, `
		Promise.resolve(1)
			.then(value => { throw new Error("failed"); })
			.catch(error => error);
	`)
	stmt := requireType[*ast.ExpressionStatement](t, program.Statements[0])
	catchCall := requireType[*ast.InvokeExpression](t, stmt.Expression)
	catchMember := requireType[*ast.MemberExpression](t, catchCall.Callee)
	if catchMember.Property != "catch" {
		t.Fatalf("expected catch call, got %q", catchMember.Property)
	}
	thenCall := requireType[*ast.InvokeExpression](t, catchMember.Target)
	thenMember := requireType[*ast.MemberExpression](t, thenCall.Callee)
	if thenMember.Property != "then" {
		t.Fatalf("expected then call, got %q", thenMember.Property)
	}
	handler := requireType[*ast.FunctionExpression](t, thenCall.Arguments[0])
	requireType[*ast.ThrowStatement](t, handler.Body[0])
}
