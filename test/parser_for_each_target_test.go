package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserForOfAssignmentTarget(t *testing.T) {
	program := parseProgram(t, `for (item of items) { print(item); }`)
	stmt := requireType[*ast.ForOfStatement](t, program.Statements[0])
	requireType[*ast.Identifier](t, stmt.Target)
	if stmt.Pattern != nil {
		t.Fatalf("expected assignment target head, got pattern %#v", stmt.Pattern)
	}
}

func TestParserForInIndexAssignmentTarget(t *testing.T) {
	program := parseProgram(t, `for (keys[index] in item) { index += 1; }`)
	stmt := requireType[*ast.ForInStatement](t, program.Statements[0])
	requireType[*ast.IndexExpression](t, stmt.Target)
	if stmt.Pattern != nil {
		t.Fatalf("expected assignment target head, got pattern %#v", stmt.Pattern)
	}
}
