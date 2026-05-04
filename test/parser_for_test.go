package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserForStatement(t *testing.T) {
	program := parseProgram(t, `for (var i = 0; i < limit; i += 1) { total += i; }`)
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}
	stmt := requireType[*ast.ForStatement](t, program.Statements[0])
	requireType[*ast.VariableDecl](t, stmt.Init)
	requireType[*ast.ComparisonExpression](t, stmt.Condition)
	requireType[*ast.AssignmentStatement](t, stmt.Update)
	if len(stmt.Body) != 1 {
		t.Fatalf("expected 1 body statement, got %d", len(stmt.Body))
	}
	requireType[*ast.AssignmentStatement](t, stmt.Body[0])
}

func TestParserForStatementEmptyClauses(t *testing.T) {
	program := parseProgram(t, `for (;;) { break; }`)
	stmt := requireType[*ast.ForStatement](t, program.Statements[0])
	if stmt.Init != nil || stmt.Condition != nil || stmt.Update != nil {
		t.Fatalf("expected empty for clauses, got %#v", stmt)
	}
	requireType[*ast.BreakStatement](t, stmt.Body[0])
}

func TestParserForStatementExpressionClauses(t *testing.T) {
	program := parseProgram(t, `for (start(); ready; step()) { continue; }`)
	stmt := requireType[*ast.ForStatement](t, program.Statements[0])
	requireType[*ast.ExpressionStatement](t, stmt.Init)
	requireType[*ast.Identifier](t, stmt.Condition)
	requireType[*ast.ExpressionStatement](t, stmt.Update)
	requireType[*ast.ContinueStatement](t, stmt.Body[0])
}
