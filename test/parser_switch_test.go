package test

import (
	"testing"

	"jayess-go/ast"
	"jayess-go/lexer"
	"jayess-go/parser"
)

func TestParserSwitchStatement(t *testing.T) {
	program := parseProgram(t, `switch (kind) { case "a": value = 1; break; default: value = 0; }`)
	stmt := requireType[*ast.SwitchStatement](t, program.Statements[0])
	requireType[*ast.Identifier](t, stmt.Discriminant)
	if len(stmt.Cases) != 1 {
		t.Fatalf("expected 1 switch case, got %d", len(stmt.Cases))
	}
	requireType[*ast.StringLiteral](t, stmt.Cases[0].Test)
	if len(stmt.Cases[0].Consequent) != 2 {
		t.Fatalf("expected assignment and break in case consequent, got %d", len(stmt.Cases[0].Consequent))
	}
	requireType[*ast.AssignmentStatement](t, stmt.Cases[0].Consequent[0])
	requireType[*ast.BreakStatement](t, stmt.Cases[0].Consequent[1])
	if len(stmt.Default) != 1 {
		t.Fatalf("expected 1 default statement, got %d", len(stmt.Default))
	}
}

func TestParserSwitchFallthroughCase(t *testing.T) {
	program := parseProgram(t, `switch (value) { case 1: case 2: result = 2; }`)
	stmt := requireType[*ast.SwitchStatement](t, program.Statements[0])
	if len(stmt.Cases) != 2 {
		t.Fatalf("expected 2 switch cases, got %d", len(stmt.Cases))
	}
	if len(stmt.Cases[0].Consequent) != 0 {
		t.Fatalf("expected empty first case consequent, got %d", len(stmt.Cases[0].Consequent))
	}
	if len(stmt.Cases[1].Consequent) != 1 {
		t.Fatalf("expected second case consequent, got %d", len(stmt.Cases[1].Consequent))
	}
}

func TestParserSwitchRejectsUnexpectedToken(t *testing.T) {
	_, err := parser.New(lexer.New(`switch (value) { value = 1; }`)).ParseProgram()
	if err == nil {
		t.Fatalf("expected switch clause error")
	}
}

func TestParserSwitchRejectsDuplicateDefault(t *testing.T) {
	_, err := parser.New(lexer.New(`switch (value) { default: value = 1; default: value = 2; }`)).ParseProgram()
	if err == nil {
		t.Fatalf("expected duplicate default error")
	}
}
