package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserObjectAsyncMethod(t *testing.T) {
	expr := parseExpression(t, `({ async load(read) { return await read(); } })`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	method := requireType[*ast.FunctionExpression](t, object.Properties[0].Value)
	if method.Name != "load" || !method.IsAsync || method.IsGenerator {
		t.Fatalf("unexpected async object method: %#v", method)
	}
}

func TestParserObjectGeneratorMethod(t *testing.T) {
	expr := parseExpression(t, `({ *values(value) { yield value; } })`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	method := requireType[*ast.FunctionExpression](t, object.Properties[0].Value)
	if method.Name != "values" || !method.IsGenerator || method.IsAsync {
		t.Fatalf("unexpected generator object method: %#v", method)
	}
}

func TestParserObjectAsyncGeneratorComputedMethod(t *testing.T) {
	expr := parseExpression(t, `({ async *[name](value) { yield await value; } })`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	property := object.Properties[0]
	method := requireType[*ast.FunctionExpression](t, property.Value)
	if !property.Computed || !method.IsAsync || !method.IsGenerator {
		t.Fatalf("unexpected computed async generator object method: %#v", property)
	}
}

func TestParserObjectMethodNamedAsync(t *testing.T) {
	expr := parseExpression(t, `({ async() { return 1; } })`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	method := requireType[*ast.FunctionExpression](t, object.Properties[0].Value)
	if method.Name != "async" || method.IsAsync {
		t.Fatalf("expected object method named async, got %#v", method)
	}
}

func TestParserObjectAsyncMethodRejectsLineBreakAfterAsync(t *testing.T) {
	_, err := parseProgramError(`
		const item = {
			async
			load() {}
		};
	`)
	if err == nil {
		t.Fatalf("expected line break after async to reject object async method")
	}
}
