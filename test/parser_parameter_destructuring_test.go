package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserFunctionDestructuringParameters(t *testing.T) {
	program := parseProgram(t, `function read({ name }, [count]) { return name; }`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[0])
	if len(fn.Params) != 2 {
		t.Fatalf("expected 2 params, got %#v", fn.Params)
	}
	requireType[*ast.ObjectBindingPattern](t, fn.Params[0].Pattern)
	requireType[*ast.ArrayBindingPattern](t, fn.Params[1].Pattern)
}

func TestParserArrowDestructuringParameter(t *testing.T) {
	expr := parseExpression(t, `({ name }) => name`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if len(fn.Params) != 1 {
		t.Fatalf("expected 1 param, got %#v", fn.Params)
	}
	requireType[*ast.ObjectBindingPattern](t, fn.Params[0].Pattern)
}

func TestParserSingleArrowParameterHasBindingPattern(t *testing.T) {
	expr := parseExpression(t, `value => value`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	param := requireType[*ast.BindingName](t, fn.Params[0].Pattern)
	if param.Name != "value" {
		t.Fatalf("expected binding name value, got %q", param.Name)
	}
}

func TestParserDestructuringParameterDefault(t *testing.T) {
	program := parseProgram(t, `function read({ name } = fallback) { return name; }`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[0])
	if fn.Params[0].Default == nil {
		t.Fatalf("expected parameter default")
	}
	requireType[*ast.ObjectBindingPattern](t, fn.Params[0].Pattern)
}
