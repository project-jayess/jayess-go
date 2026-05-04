package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserWhileStatement(t *testing.T) {
	program := parseProgram(t, `while (ready) { count += 1; }`)
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}
	stmt := requireType[*ast.WhileStatement](t, program.Statements[0])
	requireType[*ast.Identifier](t, stmt.Condition)
	if len(stmt.Body) != 1 {
		t.Fatalf("expected 1 while body statement, got %d", len(stmt.Body))
	}
	requireType[*ast.AssignmentStatement](t, stmt.Body[0])
}

func TestParserBreakAndContinue(t *testing.T) {
	program := parseProgram(t, `while (running) { break outer; continue; }`)
	stmt := requireType[*ast.WhileStatement](t, program.Statements[0])
	if len(stmt.Body) != 2 {
		t.Fatalf("expected 2 while body statements, got %d", len(stmt.Body))
	}
	br := requireType[*ast.BreakStatement](t, stmt.Body[0])
	if br.Label != "outer" {
		t.Fatalf("expected break label outer, got %q", br.Label)
	}
	requireType[*ast.ContinueStatement](t, stmt.Body[1])
}

func TestParserJumpLabelMustBeSameLine(t *testing.T) {
	program := parseProgram(t, "break\nlabel;")
	if len(program.Statements) != 2 {
		t.Fatalf("expected break and expression statements, got %d", len(program.Statements))
	}
	br := requireType[*ast.BreakStatement](t, program.Statements[0])
	if br.Label != "" {
		t.Fatalf("expected unlabeled break, got %q", br.Label)
	}
}
