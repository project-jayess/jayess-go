package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserAnonymousFunctionExpression(t *testing.T) {
	program := parseProgram(t, `const fn = function (value) { return value; };`)
	decl := requireType[*ast.VariableDecl](t, program.Statements[0])
	fn := requireType[*ast.FunctionExpression](t, decl.Value)
	if fn.Name != "" {
		t.Fatalf("expected anonymous function expression, got name %q", fn.Name)
	}
	if len(fn.Params) != 1 || fn.Params[0].Name != "value" {
		t.Fatalf("unexpected params: %#v", fn.Params)
	}
	requireType[*ast.ReturnStatement](t, fn.Body[0])
}

func TestParserNamedFunctionExpression(t *testing.T) {
	expr := parseExpression(t, `function recur(value) { return value; }`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if fn.Name != "recur" {
		t.Fatalf("expected function expression name recur, got %q", fn.Name)
	}
}

func TestParserFunctionExpressionAsArgument(t *testing.T) {
	expr := parseExpression(t, `map(function (item) { return item; })`)
	call := requireType[*ast.CallExpression](t, expr)
	if len(call.Arguments) != 1 {
		t.Fatalf("expected one argument, got %d", len(call.Arguments))
	}
	requireType[*ast.FunctionExpression](t, call.Arguments[0])
}
