package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
)

func TestParserSingleParamArrowFunction(t *testing.T) {
	expr := parseExpression(t, `value => value + 1`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if !fn.IsArrowFunction {
		t.Fatalf("expected arrow function")
	}
	if len(fn.Params) != 1 || fn.Params[0].Name != "value" {
		t.Fatalf("unexpected params: %#v", fn.Params)
	}
	requireType[*ast.BinaryExpression](t, fn.ExpressionBody)
}

func TestParserParenthesizedArrowFunction(t *testing.T) {
	expr := parseExpression(t, `(left, right) => left + right`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if len(fn.Params) != 2 {
		t.Fatalf("expected 2 params, got %#v", fn.Params)
	}
	requireType[*ast.BinaryExpression](t, fn.ExpressionBody)
}

func TestParserArrowFunctionBlockBody(t *testing.T) {
	expr := parseExpression(t, `() => { return 1; }`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if len(fn.Params) != 0 {
		t.Fatalf("expected no params, got %#v", fn.Params)
	}
	if fn.ExpressionBody != nil || len(fn.Body) != 1 {
		t.Fatalf("expected block body, got %#v", fn)
	}
	requireType[*ast.ReturnStatement](t, fn.Body[0])
}

func TestParserArrowFunctionAsArgument(t *testing.T) {
	expr := parseExpression(t, `map(values, item => item.name)`)
	call := requireType[*ast.CallExpression](t, expr)
	if len(call.Arguments) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(call.Arguments))
	}
	requireType[*ast.FunctionExpression](t, call.Arguments[1])
}

func TestParserRejectsLineBreakBeforeSingleParamArrow(t *testing.T) {
	_, err := parseProgramError(`
		value
		=> value;
	`)
	if err == nil {
		t.Fatalf("expected line break before arrow to fail")
	}
	if !strings.Contains(err.Error(), "line terminator before arrow is not allowed") {
		t.Fatalf("expected clear arrow line terminator diagnostic, got %v", err)
	}
}

func TestParserRejectsLineBreakBeforeParenthesizedArrow(t *testing.T) {
	_, err := parseProgramError(`
		(value)
		=> value;
	`)
	if err == nil {
		t.Fatalf("expected line break before parenthesized arrow to fail")
	}
	if !strings.Contains(err.Error(), "line terminator before arrow is not allowed") {
		t.Fatalf("expected clear arrow line terminator diagnostic, got %v", err)
	}
}
