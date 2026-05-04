package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserObjectMethodDefinition(t *testing.T) {
	expr := parseExpression(t, `{ identity(value) { return value; } }`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	if len(object.Properties) != 1 {
		t.Fatalf("expected one property, got %d", len(object.Properties))
	}
	property := object.Properties[0]
	if property.Key != "identity" || !property.Method {
		t.Fatalf("unexpected method property: %#v", property)
	}
	fn := requireType[*ast.FunctionExpression](t, property.Value)
	if fn.Name != "identity" {
		t.Fatalf("expected function name identity, got %q", fn.Name)
	}
	if len(fn.Params) != 1 || fn.Params[0].Name != "value" {
		t.Fatalf("unexpected method params: %#v", fn.Params)
	}
	requireType[*ast.ReturnStatement](t, fn.Body[0])
}

func TestParserComputedObjectMethodDefinition(t *testing.T) {
	expr := parseExpression(t, `{ [methodName](value) { return value; } }`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	if len(object.Properties) != 1 {
		t.Fatalf("expected one property, got %d", len(object.Properties))
	}
	property := object.Properties[0]
	if !property.Computed || !property.Method {
		t.Fatalf("expected computed method property: %#v", property)
	}
	requireType[*ast.Identifier](t, property.KeyExpr)
	requireType[*ast.FunctionExpression](t, property.Value)
}

func TestParserKeywordObjectMethodDefinition(t *testing.T) {
	expr := parseExpression(t, `{ default(value) { return value; } }`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	property := object.Properties[0]
	if property.Key != "default" || !property.Method {
		t.Fatalf("unexpected keyword method property: %#v", property)
	}
}
