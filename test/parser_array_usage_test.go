package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserArrayIndexAssignment(t *testing.T) {
	program := parseProgram(t, `values[0] = next;`)
	stmt := requireType[*ast.AssignmentStatement](t, program.Statements[0])
	requireType[*ast.IndexExpression](t, stmt.Target)
	requireType[*ast.Identifier](t, stmt.Value)
}

func TestParserArrayLengthMemberAccess(t *testing.T) {
	expr := parseExpression(t, `values.length`)
	member := requireType[*ast.MemberExpression](t, expr)
	if member.Property != "length" {
		t.Fatalf("expected length member access, got %q", member.Property)
	}
	requireType[*ast.Identifier](t, member.Target)
}

func TestParserArrayIterationForOf(t *testing.T) {
	program := parseProgram(t, `for (const value of values) { print(value); }`)
	stmt := requireType[*ast.ForOfStatement](t, program.Statements[0])
	requireType[*ast.Identifier](t, stmt.Iterable)
}
