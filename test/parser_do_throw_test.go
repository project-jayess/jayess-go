package test

import (
	"testing"

	"jayess-go/ast"
	"jayess-go/lexer"
	"jayess-go/parser"
)

func TestParserDoWhileStatement(t *testing.T) {
	program := parseProgram(t, `do { count += 1; } while (count < limit);`)
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}
	stmt := requireType[*ast.DoWhileStatement](t, program.Statements[0])
	if len(stmt.Body) != 1 {
		t.Fatalf("expected 1 do body statement, got %d", len(stmt.Body))
	}
	requireType[*ast.AssignmentStatement](t, stmt.Body[0])
	requireType[*ast.ComparisonExpression](t, stmt.Condition)
}

func TestParserThrowStatement(t *testing.T) {
	program := parseProgram(t, `throw makeError("failed");`)
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}
	stmt := requireType[*ast.ThrowStatement](t, program.Statements[0])
	requireType[*ast.CallExpression](t, stmt.Value)
}

func TestParserThrowRequiresSameLineExpression(t *testing.T) {
	_, err := parser.New(lexer.New("throw\nmakeError();")).ParseProgram()
	if err == nil {
		t.Fatalf("expected throw line-break error")
	}
}
