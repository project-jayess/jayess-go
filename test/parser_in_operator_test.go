package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserInOperator(t *testing.T) {
	expr := parseExpression(t, `"name" in item`)
	compare := requireType[*ast.ComparisonExpression](t, expr)
	if compare.Operator != ast.OperatorIn {
		t.Fatalf("expected in operator, got %q", compare.Operator)
	}
	requireType[*ast.StringLiteral](t, compare.Left)
	right := requireType[*ast.Identifier](t, compare.Right)
	if right.Name != "item" {
		t.Fatalf("expected right identifier item, got %q", right.Name)
	}
}

func TestParserInOperatorPrecedence(t *testing.T) {
	expr := parseExpression(t, `"value" in item && ready`)
	logical := requireType[*ast.LogicalExpression](t, expr)
	requireType[*ast.ComparisonExpression](t, logical.Left)
	requireType[*ast.Identifier](t, logical.Right)
}
