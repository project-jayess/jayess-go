package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserArrayLiteral(t *testing.T) {
	expr := parseExpression(t, "[1, value, ...rest]")
	array := requireType[*ast.ArrayLiteral](t, expr)
	if len(array.Elements) != 3 {
		t.Fatalf("expected 3 array elements, got %d", len(array.Elements))
	}
	requireType[*ast.NumberLiteral](t, array.Elements[0])
	requireType[*ast.Identifier](t, array.Elements[1])
	requireType[*ast.SpreadExpression](t, array.Elements[2])
}

func TestParserArrayLiteralElisions(t *testing.T) {
	expr := parseExpression(t, "[first, , third]")
	array := requireType[*ast.ArrayLiteral](t, expr)
	if len(array.Elements) != 3 {
		t.Fatalf("expected 3 array elements, got %d", len(array.Elements))
	}
	if array.Elements[1] != nil {
		t.Fatalf("expected skipped middle element, got %#v", array.Elements[1])
	}
	requireType[*ast.Identifier](t, array.Elements[0])
	requireType[*ast.Identifier](t, array.Elements[2])
}

func TestParserArrayLiteralLeadingElisions(t *testing.T) {
	expr := parseExpression(t, "[, , value]")
	array := requireType[*ast.ArrayLiteral](t, expr)
	if len(array.Elements) != 3 {
		t.Fatalf("expected 3 array elements, got %d", len(array.Elements))
	}
	if array.Elements[0] != nil || array.Elements[1] != nil {
		t.Fatalf("expected skipped leading elements, got %#v", array.Elements)
	}
	requireType[*ast.Identifier](t, array.Elements[2])
}

func TestParserObjectLiteral(t *testing.T) {
	expr := parseExpression(t, `{ name: "Jay", count: 2, ...extra }`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	if len(object.Properties) != 3 {
		t.Fatalf("expected 3 object properties, got %d", len(object.Properties))
	}
	if object.Properties[0].Key != "name" {
		t.Fatalf("expected first key name, got %q", object.Properties[0].Key)
	}
	if !object.Properties[2].Spread {
		t.Fatalf("expected third property to be spread")
	}
}

func TestParserObjectComputedProperty(t *testing.T) {
	expr := parseExpression(t, `{ [prefix + name]: value }`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	if len(object.Properties) != 1 {
		t.Fatalf("expected 1 object property, got %d", len(object.Properties))
	}
	property := object.Properties[0]
	if !property.Computed {
		t.Fatalf("expected computed object property")
	}
	requireType[*ast.BinaryExpression](t, property.KeyExpr)
	requireType[*ast.Identifier](t, property.Value)
}

func TestParserObjectKeywordPropertyName(t *testing.T) {
	expr := parseExpression(t, `{ default: value, class: "Widget" }`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	if len(object.Properties) != 2 {
		t.Fatalf("expected 2 object properties, got %d", len(object.Properties))
	}
	if object.Properties[0].Key != "default" || object.Properties[1].Key != "class" {
		t.Fatalf("unexpected keyword properties: %#v", object.Properties)
	}
}

func TestParserCollectionTrailingCommas(t *testing.T) {
	requireType[*ast.ArrayLiteral](t, parseExpression(t, "[1,]"))
	requireType[*ast.ObjectLiteral](t, parseExpression(t, "{ value: 1, }"))
}
