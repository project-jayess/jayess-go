package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserCustomIterableObjectMethod(t *testing.T) {
	expr := parseExpression(t, `({ [Symbol.iterator]() { return iterator; } })`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	if len(object.Properties) != 1 {
		t.Fatalf("expected one property, got %d", len(object.Properties))
	}
	property := object.Properties[0]
	if !property.Computed || !property.Method {
		t.Fatalf("expected computed method property: %#v", property)
	}
	member := requireType[*ast.MemberExpression](t, property.KeyExpr)
	if member.Property != "iterator" {
		t.Fatalf("expected iterator property, got %q", member.Property)
	}
	fn := requireType[*ast.FunctionExpression](t, property.Value)
	requireType[*ast.ReturnStatement](t, fn.Body[0])
}

func TestParserIteratorProtocolNextMethodShape(t *testing.T) {
	expr := parseExpression(t, `({ next() { return { value: current, done: false }; } })`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	property := object.Properties[0]
	if property.Key != "next" || !property.Method {
		t.Fatalf("expected iterator next method, got %#v", property)
	}
	fn := requireType[*ast.FunctionExpression](t, property.Value)
	ret := requireType[*ast.ReturnStatement](t, fn.Body[0])
	result := requireType[*ast.ObjectLiteral](t, ret.Value)
	if len(result.Properties) != 2 {
		t.Fatalf("expected value and done properties, got %d", len(result.Properties))
	}
}

func TestParserForOfCustomIterableCall(t *testing.T) {
	program := parseProgram(t, `for (const value of makeIterable()) { print(value); }`)
	stmt := requireType[*ast.ForOfStatement](t, program.Statements[0])
	if stmt.Name != "value" {
		t.Fatalf("expected loop binding value, got %q", stmt.Name)
	}
	requireType[*ast.CallExpression](t, stmt.Iterable)
}
