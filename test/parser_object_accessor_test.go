package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserObjectGettersAndSetters(t *testing.T) {
	expr := parseExpression(t, `{ get value() { return this.count; }, set value(next) { this.count = next; } }`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	if len(object.Properties) != 2 {
		t.Fatalf("expected two accessors, got %d", len(object.Properties))
	}
	if !object.Properties[0].Getter || object.Properties[0].Key != "value" {
		t.Fatalf("expected value getter, got %#v", object.Properties[0])
	}
	if !object.Properties[1].Setter || len(requireType[*ast.FunctionExpression](t, object.Properties[1].Value).Params) != 1 {
		t.Fatalf("expected value setter with one parameter, got %#v", object.Properties[1])
	}
}

func TestParserRejectsObjectGetterWithParameter(t *testing.T) {
	_, err := parseProgramError(`const item = { get value(next) { return next; } };`)
	if err == nil {
		t.Fatalf("expected object getter parameter error")
	}
}

func TestParserRejectsObjectSetterWithoutParameter(t *testing.T) {
	_, err := parseProgramError(`const item = { set value() {} };`)
	if err == nil {
		t.Fatalf("expected object setter parameter error")
	}
}

func TestParserRejectsObjectSetterRestParameter(t *testing.T) {
	_, err := parseProgramError(`const item = { set value(...next) {} };`)
	if err == nil {
		t.Fatalf("expected object setter rest parameter error")
	}
}

func TestParserComputedObjectGettersAndSetters(t *testing.T) {
	expr := parseExpression(t, `{ get [key]() { return this.count; }, set [key](next) { this.count = next; } }`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	if len(object.Properties) != 2 {
		t.Fatalf("expected two computed accessors, got %d", len(object.Properties))
	}
	if !object.Properties[0].Computed || !object.Properties[0].Getter {
		t.Fatalf("expected computed getter, got %#v", object.Properties[0])
	}
	requireType[*ast.Identifier](t, object.Properties[0].KeyExpr)
	if !object.Properties[1].Computed || !object.Properties[1].Setter {
		t.Fatalf("expected computed setter, got %#v", object.Properties[1])
	}
}

func TestParserRejectsComputedObjectGetterWithParameter(t *testing.T) {
	_, err := parseProgramError(`const item = { get [key](next) { return next; } };`)
	if err == nil {
		t.Fatalf("expected computed object getter parameter error")
	}
}

func TestParserRejectsComputedObjectSetterRestParameter(t *testing.T) {
	_, err := parseProgramError(`const item = { set [key](...next) {} };`)
	if err == nil {
		t.Fatalf("expected computed object setter rest parameter error")
	}
}
