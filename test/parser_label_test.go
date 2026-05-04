package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserLabeledStatement(t *testing.T) {
	program := parseProgram(t, `outer: while (ready) { break outer; }`)
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}
	label := requireType[*ast.LabeledStatement](t, program.Statements[0])
	if label.Label != "outer" {
		t.Fatalf("expected label outer, got %q", label.Label)
	}
	loop := requireType[*ast.WhileStatement](t, label.Statement)
	br := requireType[*ast.BreakStatement](t, loop.Body[0])
	if br.Label != "outer" {
		t.Fatalf("expected break outer, got %q", br.Label)
	}
}

func TestParserNestedLabeledBlocks(t *testing.T) {
	program := parseProgram(t, `retry: { continue retry; }`)
	label := requireType[*ast.LabeledStatement](t, program.Statements[0])
	block := requireType[*ast.BlockStatement](t, label.Statement)
	cont := requireType[*ast.ContinueStatement](t, block.Statements[0])
	if cont.Label != "retry" {
		t.Fatalf("expected continue retry, got %q", cont.Label)
	}
}

func TestParserIdentifierExpressionStatementIsNotLabel(t *testing.T) {
	program := parseProgram(t, `ready;`)
	requireType[*ast.ExpressionStatement](t, program.Statements[0])
}
