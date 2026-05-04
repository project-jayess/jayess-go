package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserBlockAndReturnStatements(t *testing.T) {
	program := parseProgram(t, "{ var value = 1; return value }")
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 top-level block, got %d statements", len(program.Statements))
	}
	block := requireType[*ast.BlockStatement](t, program.Statements[0])
	if len(block.Statements) != 2 {
		t.Fatalf("expected 2 block statements, got %d", len(block.Statements))
	}
	ret := requireType[*ast.ReturnStatement](t, block.Statements[1])
	requireType[*ast.Identifier](t, ret.Value)
}

func TestParserReturnWithoutValue(t *testing.T) {
	program := parseProgram(t, "return\nvalue;")
	if len(program.Statements) != 2 {
		t.Fatalf("expected return and expression statements, got %d", len(program.Statements))
	}
	ret := requireType[*ast.ReturnStatement](t, program.Statements[0])
	if ret.Value != nil {
		t.Fatalf("expected bare return, got %#v", ret.Value)
	}
}

func TestParserIfElseStatement(t *testing.T) {
	program := parseProgram(t, `if (ready) { return 1; } else { return 0; }`)
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 if statement, got %d", len(program.Statements))
	}
	stmt := requireType[*ast.IfStatement](t, program.Statements[0])
	requireType[*ast.Identifier](t, stmt.Condition)
	if len(stmt.Consequence) != 1 || len(stmt.Alternative) != 1 {
		t.Fatalf("expected consequence and alternative, got %#v", stmt)
	}
	requireType[*ast.ReturnStatement](t, stmt.Consequence[0])
	requireType[*ast.ReturnStatement](t, stmt.Alternative[0])
}

func TestParserElseIfStatement(t *testing.T) {
	program := parseProgram(t, `if (first) { return 1; } else if (second) { return 2; }`)
	stmt := requireType[*ast.IfStatement](t, program.Statements[0])
	if len(stmt.Alternative) != 1 {
		t.Fatalf("expected else-if alternative, got %#v", stmt.Alternative)
	}
	requireType[*ast.IfStatement](t, stmt.Alternative[0])
}
