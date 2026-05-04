package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserSpreadCallArgument(t *testing.T) {
	expr := parseExpression(t, `call(first, ...rest, last)`)
	call := requireType[*ast.CallExpression](t, expr)
	if len(call.Arguments) != 3 {
		t.Fatalf("expected 3 arguments, got %d", len(call.Arguments))
	}
	requireType[*ast.Identifier](t, call.Arguments[0])
	spread := requireType[*ast.SpreadExpression](t, call.Arguments[1])
	if ident := requireType[*ast.Identifier](t, spread.Value); ident.Name != "rest" {
		t.Fatalf("expected rest spread value, got %q", ident.Name)
	}
}

func TestParserSpreadInvokeArgument(t *testing.T) {
	expr := parseExpression(t, `target.method(...args)`)
	call := requireType[*ast.InvokeExpression](t, expr)
	if len(call.Arguments) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(call.Arguments))
	}
	requireType[*ast.SpreadExpression](t, call.Arguments[0])
}

func TestParserTrailingCommaCallArguments(t *testing.T) {
	expr := parseExpression(t, `call(first, second,)`)
	call := requireType[*ast.CallExpression](t, expr)
	if len(call.Arguments) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(call.Arguments))
	}
}

func TestParserTrailingCommaInvokeArguments(t *testing.T) {
	expr := parseExpression(t, `target.method(first, second,)`)
	call := requireType[*ast.InvokeExpression](t, expr)
	if len(call.Arguments) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(call.Arguments))
	}
}

func TestParserSpreadCallInsideArrow(t *testing.T) {
	expr := parseExpression(t, `(...args) => call(...args)`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if len(fn.Params) != 1 || !fn.Params[0].Rest {
		t.Fatalf("expected rest parameter, got %#v", fn.Params)
	}
	call := requireType[*ast.CallExpression](t, fn.ExpressionBody)
	requireType[*ast.SpreadExpression](t, call.Arguments[0])
}
