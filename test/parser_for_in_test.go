package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserForInStatement(t *testing.T) {
	program := parseProgram(t, `for (const key in item) { print(key); }`)
	stmt := requireType[*ast.ForInStatement](t, program.Statements[0])
	if stmt.Kind != ast.DeclarationConst || stmt.Name != "key" {
		t.Fatalf("unexpected for...in binding: %#v", stmt)
	}
	requireType[*ast.Identifier](t, stmt.Object)
	requireType[*ast.ExpressionStatement](t, stmt.Body[0])
}

func TestParserForInVarBinding(t *testing.T) {
	program := parseProgram(t, `for (var key in item) { continue; }`)
	stmt := requireType[*ast.ForInStatement](t, program.Statements[0])
	if stmt.Kind != ast.DeclarationVar || stmt.Name != "key" {
		t.Fatalf("unexpected for...in binding: %#v", stmt)
	}
	requireType[*ast.ContinueStatement](t, stmt.Body[0])
}

func TestParserForInObjectBindingPattern(t *testing.T) {
	program := parseProgram(t, `for (const { key } in item) { print(key); }`)
	stmt := requireType[*ast.ForInStatement](t, program.Statements[0])
	pattern := requireType[*ast.ObjectBindingPattern](t, stmt.Pattern)
	if len(pattern.Properties) != 1 {
		t.Fatalf("expected one object binding property, got %d", len(pattern.Properties))
	}
}

func TestParserRejectsForInBindingInitializer(t *testing.T) {
	_, err := parseProgramError(`for (const key = "name" in item) { print(key); }`)
	if err == nil {
		t.Fatalf("expected for...in binding initializer error")
	}
}
