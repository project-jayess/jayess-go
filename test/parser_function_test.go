package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserFunctionDeclaration(t *testing.T) {
	program := parseProgram(t, `function main(args, extra) { const value = args; return value; }`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[0])
	if fn.Name != "main" {
		t.Fatalf("expected function main, got %q", fn.Name)
	}
	if len(fn.Params) != 2 || fn.Params[0].Name != "args" || fn.Params[1].Name != "extra" {
		t.Fatalf("unexpected params: %#v", fn.Params)
	}
	if len(fn.Body) != 2 {
		t.Fatalf("expected 2 body statements, got %d", len(fn.Body))
	}
	requireType[*ast.VariableDecl](t, fn.Body[0])
	requireType[*ast.ReturnStatement](t, fn.Body[1])
}

func TestParserFunctionDeclarationWithoutParams(t *testing.T) {
	program := parseProgram(t, `function ready() { return true; }`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[0])
	if len(fn.Params) != 0 {
		t.Fatalf("expected no params, got %#v", fn.Params)
	}
}

func TestParserNestedFunctionDeclaration(t *testing.T) {
	program := parseProgram(t, `function outer() { function inner(value) { return value; } return 0; }`)
	outer := requireType[*ast.FunctionDecl](t, program.Statements[0])
	inner := requireType[*ast.FunctionDecl](t, outer.Body[0])
	if inner.Name != "inner" || len(inner.Params) != 1 {
		t.Fatalf("unexpected nested function: %#v", inner)
	}
	requireType[*ast.ReturnStatement](t, outer.Body[1])
}
