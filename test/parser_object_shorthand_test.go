package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserObjectShorthandProperty(t *testing.T) {
	expr := parseExpression(t, `{ name, count: total }`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	if len(object.Properties) != 2 {
		t.Fatalf("expected two properties, got %d", len(object.Properties))
	}
	property := object.Properties[0]
	if !property.Shorthand || property.Key != "name" {
		t.Fatalf("expected shorthand name property, got %#v", property)
	}
	value := requireType[*ast.Identifier](t, property.Value)
	if value.Name != "name" {
		t.Fatalf("expected shorthand identifier name, got %q", value.Name)
	}
}
