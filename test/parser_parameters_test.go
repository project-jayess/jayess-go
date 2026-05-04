package test

import (
	"testing"

	"jayess-go/ast"
	"jayess-go/lexer"
	"jayess-go/parser"
)

func TestParserDefaultAndRestParameters(t *testing.T) {
	program := parseProgram(t, `function collect(first = 1, ...rest) { return rest; }`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[0])
	if len(fn.Params) != 2 {
		t.Fatalf("expected 2 params, got %#v", fn.Params)
	}
	if fn.Params[0].Default == nil {
		t.Fatalf("expected first parameter default value")
	}
	if !fn.Params[1].Rest || fn.Params[1].Name != "rest" {
		t.Fatalf("expected rest parameter, got %#v", fn.Params[1])
	}
}

func TestParserArrowDefaultParameter(t *testing.T) {
	expr := parseExpression(t, `(value = 1) => value`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if fn.Params[0].Default == nil {
		t.Fatalf("expected arrow parameter default")
	}
	requireType[*ast.Identifier](t, fn.ExpressionBody)
}

func TestParserTrailingCommaParameters(t *testing.T) {
	program := parseProgram(t, `function sum(first, second,) { return first + second; }`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[0])
	if len(fn.Params) != 2 {
		t.Fatalf("expected 2 params, got %#v", fn.Params)
	}
}

func TestParserTrailingCommaArrowParameters(t *testing.T) {
	expr := parseExpression(t, `(first, second,) => first + second`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if len(fn.Params) != 2 {
		t.Fatalf("expected 2 params, got %#v", fn.Params)
	}
}

func TestParserRejectsRestParameterBeforeEnd(t *testing.T) {
	_, err := parser.New(lexer.New(`function bad(...rest, next) {}`)).ParseProgram()
	if err == nil {
		t.Fatalf("expected rest parameter ordering error")
	}
}

func TestParserRejectsRestParameterDefault(t *testing.T) {
	_, err := parser.New(lexer.New(`function bad(...rest = []) {}`)).ParseProgram()
	if err == nil {
		t.Fatalf("expected rest parameter default error")
	}
}
