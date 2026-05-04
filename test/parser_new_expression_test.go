package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserNewExpressionWithArguments(t *testing.T) {
	expr := parseExpression(t, `new Widget(1, name)`)
	newExpr := requireType[*ast.NewExpression](t, expr)
	callee := requireType[*ast.Identifier](t, newExpr.Callee)
	if callee.Name != "Widget" {
		t.Fatalf("expected Widget constructor, got %q", callee.Name)
	}
	if len(newExpr.Arguments) != 2 {
		t.Fatalf("expected two constructor args, got %d", len(newExpr.Arguments))
	}
}

func TestParserNewExpressionWithSpreadArguments(t *testing.T) {
	expr := parseExpression(t, `new Widget(first, ...rest)`)
	newExpr := requireType[*ast.NewExpression](t, expr)
	if len(newExpr.Arguments) != 2 {
		t.Fatalf("expected two constructor args, got %d", len(newExpr.Arguments))
	}
	requireType[*ast.Identifier](t, newExpr.Arguments[0])
	requireType[*ast.SpreadExpression](t, newExpr.Arguments[1])
}

func TestParserNewExpressionWithMemberCallee(t *testing.T) {
	expr := parseExpression(t, `new ns.Widget(value)`)
	newExpr := requireType[*ast.NewExpression](t, expr)
	member := requireType[*ast.MemberExpression](t, newExpr.Callee)
	if member.Property != "Widget" {
		t.Fatalf("expected Widget property, got %q", member.Property)
	}
	if len(newExpr.Arguments) != 1 {
		t.Fatalf("expected one constructor arg, got %d", len(newExpr.Arguments))
	}
}

func TestParserNewExpressionWithoutArguments(t *testing.T) {
	expr := parseExpression(t, `new Widget`)
	newExpr := requireType[*ast.NewExpression](t, expr)
	requireType[*ast.Identifier](t, newExpr.Callee)
	if len(newExpr.Arguments) != 0 {
		t.Fatalf("expected no constructor args, got %d", len(newExpr.Arguments))
	}
}

func TestParserNewTargetExpression(t *testing.T) {
	expr := parseExpression(t, `new.target`)
	requireType[*ast.NewTargetExpression](t, expr)
}

func TestParserNewTargetMemberExpression(t *testing.T) {
	expr := parseExpression(t, `new.target.name`)
	member := requireType[*ast.MemberExpression](t, expr)
	requireType[*ast.NewTargetExpression](t, member.Target)
	if member.Property != "name" {
		t.Fatalf("expected new.target.name property, got %q", member.Property)
	}
}

func TestParserRejectsNewExpressionWithOptionalMemberCallee(t *testing.T) {
	_, err := parseProgramError(`const value = new ns?.Widget();`)
	if err == nil {
		t.Fatalf("expected optional chain new target error")
	}
}

func TestParserRejectsNewExpressionWithOptionalCallCallee(t *testing.T) {
	_, err := parseProgramError(`const value = new Widget?.();`)
	if err == nil {
		t.Fatalf("expected optional call new target error")
	}
}
