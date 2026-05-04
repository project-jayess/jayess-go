package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserFunctionValueInArray(t *testing.T) {
	expr := parseExpression(t, `[function (value) { return value; }]`)
	array := requireType[*ast.ArrayLiteral](t, expr)
	if len(array.Elements) != 1 {
		t.Fatalf("expected one array element, got %d", len(array.Elements))
	}
	requireType[*ast.FunctionExpression](t, array.Elements[0])
}

func TestParserFunctionValueInObject(t *testing.T) {
	expr := parseExpression(t, `({ fn: function (value) { return value; } })`)
	object := requireType[*ast.ObjectLiteral](t, expr)
	if len(object.Properties) != 1 {
		t.Fatalf("expected one object property, got %d", len(object.Properties))
	}
	requireType[*ast.FunctionExpression](t, object.Properties[0].Value)
}

func TestParserReturnedClosureFunctionExpression(t *testing.T) {
	program := parseProgram(t, `
		function makeAdder(base) {
			return function (value) {
				return base + value;
			};
		}
	`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[0])
	ret := requireType[*ast.ReturnStatement](t, fn.Body[0])
	closure := requireType[*ast.FunctionExpression](t, ret.Value)
	if len(closure.Params) != 1 || closure.Params[0].Name != "value" {
		t.Fatalf("unexpected closure params: %#v", closure.Params)
	}
	innerReturn := requireType[*ast.ReturnStatement](t, closure.Body[0])
	requireType[*ast.BinaryExpression](t, innerReturn.Value)
}
