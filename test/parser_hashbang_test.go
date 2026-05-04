package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserAllowsHashbangAtStartOfProgram(t *testing.T) {
	program := parseProgram(t, "#!/usr/bin/env jayess\nconst value = 1;")
	if len(program.Statements) != 1 {
		t.Fatalf("expected one statement after hashbang, got %d", len(program.Statements))
	}
	requireType[*ast.VariableDecl](t, program.Statements[0])
}

func TestParserRejectsHashbangAfterLeadingWhitespace(t *testing.T) {
	_, err := parseProgramError(" \n#!/usr/bin/env jayess\nconst value = 1;")
	if err == nil {
		t.Fatalf("expected misplaced hashbang parse error")
	}
}
