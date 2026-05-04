package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserDeleteMemberExpression(t *testing.T) {
	expr := parseExpression(t, `delete item.value`)
	unary := requireType[*ast.UnaryExpression](t, expr)
	if unary.Operator != ast.OperatorDelete {
		t.Fatalf("expected delete operator, got %q", unary.Operator)
	}
	member := requireType[*ast.MemberExpression](t, unary.Right)
	if member.Property != "value" {
		t.Fatalf("expected value property, got %q", member.Property)
	}
}

func TestParserDeleteIndexExpression(t *testing.T) {
	expr := parseExpression(t, `delete items[index]`)
	unary := requireType[*ast.UnaryExpression](t, expr)
	if unary.Operator != ast.OperatorDelete {
		t.Fatalf("expected delete operator, got %q", unary.Operator)
	}
	requireType[*ast.IndexExpression](t, unary.Right)
}
