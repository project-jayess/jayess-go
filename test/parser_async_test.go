package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
)

func TestParserAsyncFunctionDeclaration(t *testing.T) {
	program := parseProgram(t, `async function load() { return await read(); }`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[0])
	if !fn.IsAsync || fn.Name != "load" {
		t.Fatalf("unexpected async function declaration: %#v", fn)
	}
	ret := requireType[*ast.ReturnStatement](t, fn.Body[0])
	requireType[*ast.AwaitExpression](t, ret.Value)
}

func TestParserAsyncFunctionReturnValue(t *testing.T) {
	program := parseProgram(t, `async function load() { return "ready"; }`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[0])
	ret := requireType[*ast.ReturnStatement](t, fn.Body[0])
	requireType[*ast.StringLiteral](t, ret.Value)
}

func TestParserRejectsAsyncFunctionDeclarationWithLineBreakAfterAsync(t *testing.T) {
	_, err := parseProgramError("async\nfunction load() {}")
	if err == nil {
		t.Fatalf("expected async function declaration line-break error")
	}
	if !strings.Contains(err.Error(), "line terminator after async is not allowed") {
		t.Fatalf("expected clear async line terminator diagnostic, got %v", err)
	}
}

func TestParserAsyncFunctionExpression(t *testing.T) {
	expr := parseExpression(t, `async function load() { return await read(); }`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if !fn.IsAsync || fn.Name != "load" {
		t.Fatalf("unexpected async function expression: %#v", fn)
	}
}

func TestParserRejectsAsyncFunctionExpressionWithLineBreakAfterAsync(t *testing.T) {
	_, err := parseProgramError("const load = async\nfunction load() {};")
	if err == nil {
		t.Fatalf("expected async function expression line-break error")
	}
	if !strings.Contains(err.Error(), "line terminator after async is not allowed") {
		t.Fatalf("expected clear async line terminator diagnostic, got %v", err)
	}
}

func TestParserAsyncArrowFunction(t *testing.T) {
	expr := parseExpression(t, `async value => await value`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if !fn.IsAsync || !fn.IsArrowFunction {
		t.Fatalf("unexpected async arrow function: %#v", fn)
	}
	requireType[*ast.AwaitExpression](t, fn.ExpressionBody)
}

func TestParserAsyncArrowReturnValueExpression(t *testing.T) {
	expr := parseExpression(t, `async value => value + 1`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if !fn.IsAsync || !fn.IsArrowFunction {
		t.Fatalf("unexpected async arrow function: %#v", fn)
	}
	requireType[*ast.BinaryExpression](t, fn.ExpressionBody)
}

func TestParserRejectsAsyncArrowWithLineBreakAfterAsync(t *testing.T) {
	_, err := parseProgramError("const fn = async\nvalue => value;")
	if err == nil {
		t.Fatalf("expected async arrow line-break error")
	}
	if !strings.Contains(err.Error(), "line terminator after async is not allowed") {
		t.Fatalf("expected clear async line terminator diagnostic, got %v", err)
	}
}

func TestParserRejectsAsyncParenthesizedArrowWithLineBreakAfterAsync(t *testing.T) {
	_, err := parseProgramError("const fn = async\n(value) => value;")
	if err == nil {
		t.Fatalf("expected async parenthesized arrow line-break error")
	}
	if !strings.Contains(err.Error(), "line terminator after async is not allowed") {
		t.Fatalf("expected clear async line terminator diagnostic, got %v", err)
	}
}
