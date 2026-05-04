package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserDebuggerStatement(t *testing.T) {
	program := parseProgram(t, `debugger; var value = 1;`)
	requireType[*ast.DebuggerStatement](t, program.Statements[0])
	requireType[*ast.VariableDecl](t, program.Statements[1])
}

func TestParserDebuggerKeywordPropertyName(t *testing.T) {
	expr := parseExpression(t, `item.debugger`)
	member := requireType[*ast.MemberExpression](t, expr)
	if member.Property != "debugger" {
		t.Fatalf("expected debugger property, got %q", member.Property)
	}
}
