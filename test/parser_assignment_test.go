package test

import (
	"testing"

	"jayess-go/ast"
	"jayess-go/lexer"
	"jayess-go/parser"
)

func TestParserProgramAssignmentStatements(t *testing.T) {
	program := parseProgram(t, "var total = 1;\ntotal += 2 * 3;\nready ||= fallback;")
	if len(program.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(program.Statements))
	}

	add := requireType[*ast.AssignmentStatement](t, program.Statements[1])
	if add.Operator != ast.AssignmentAddAssign {
		t.Fatalf("expected += assignment, got %q", add.Operator)
	}
	if target := requireType[*ast.Identifier](t, add.Target); target.Name != "total" {
		t.Fatalf("expected assignment target total, got %q", target.Name)
	}
	requireType[*ast.BinaryExpression](t, add.Value)

	logical := requireType[*ast.AssignmentStatement](t, program.Statements[2])
	if logical.Operator != ast.AssignmentOrAssign {
		t.Fatalf("expected ||= assignment, got %q", logical.Operator)
	}
}

func TestParserModuloAssignmentStatement(t *testing.T) {
	program := parseProgram(t, "var total = 5;\ntotal %= 2;")
	stmt := requireType[*ast.AssignmentStatement](t, program.Statements[1])
	if stmt.Operator != ast.AssignmentModAssign {
		t.Fatalf("expected %%= assignment, got %q", stmt.Operator)
	}
}

func TestParserExponentiationAssignmentStatement(t *testing.T) {
	program := parseProgram(t, "var total = 5;\ntotal **= 2;")
	stmt := requireType[*ast.AssignmentStatement](t, program.Statements[1])
	if stmt.Operator != ast.AssignmentPowAssign {
		t.Fatalf("expected **= assignment, got %q", stmt.Operator)
	}
}

func TestParserBitwiseAssignmentStatements(t *testing.T) {
	program := parseProgram(t, `
		value &= mask;
		value |= mask;
		value ^= mask;
		value <<= shift;
		value >>= shift;
		value >>>= shift;
	`)
	expected := []ast.AssignmentOperator{
		ast.AssignmentBitAndAssign,
		ast.AssignmentBitOrAssign,
		ast.AssignmentBitXorAssign,
		ast.AssignmentShlAssign,
		ast.AssignmentShrAssign,
		ast.AssignmentUShrAssign,
	}
	if len(program.Statements) != len(expected) {
		t.Fatalf("expected %d statements, got %d", len(expected), len(program.Statements))
	}
	for i, operator := range expected {
		stmt := requireType[*ast.AssignmentStatement](t, program.Statements[i])
		if stmt.Operator != operator {
			t.Fatalf("statement %d: expected %q assignment, got %q", i, operator, stmt.Operator)
		}
	}
}

func TestParserProgramExpressionStatements(t *testing.T) {
	program := parseProgram(t, "1 + 2;\ntrue && false")
	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 expression statements, got %d", len(program.Statements))
	}
	requireType[*ast.ExpressionStatement](t, program.Statements[0])
	requireType[*ast.ExpressionStatement](t, program.Statements[1])
}

func TestParserProgramRejectsInvalidAssignmentTarget(t *testing.T) {
	_, err := parser.New(lexer.New("(1 + 2) = 3;")).ParseProgram()
	if err == nil {
		t.Fatalf("expected invalid assignment target error")
	}
}
