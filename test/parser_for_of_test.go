package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserForOfStatement(t *testing.T) {
	program := parseProgram(t, `for (const item of items) { total += item; }`)
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}
	stmt := requireType[*ast.ForOfStatement](t, program.Statements[0])
	if stmt.Kind != ast.DeclarationConst || stmt.Name != "item" {
		t.Fatalf("unexpected for...of binding: %#v", stmt)
	}
	requireType[*ast.Identifier](t, stmt.Iterable)
	requireType[*ast.AssignmentStatement](t, stmt.Body[0])
}

func TestParserForOfVarBinding(t *testing.T) {
	program := parseProgram(t, `for (var value of values) { continue; }`)
	stmt := requireType[*ast.ForOfStatement](t, program.Statements[0])
	if stmt.Kind != ast.DeclarationVar || stmt.Name != "value" {
		t.Fatalf("unexpected for...of binding: %#v", stmt)
	}
	requireType[*ast.ContinueStatement](t, stmt.Body[0])
}

func TestParserClassicForStillParsesAfterForOfLookahead(t *testing.T) {
	program := parseProgram(t, `for (var i = 0; i < 3; i += 1) { break; }`)
	requireType[*ast.ForStatement](t, program.Statements[0])
}

func TestParserForOfArrayBindingPattern(t *testing.T) {
	program := parseProgram(t, `for (const [name, count] of entries) { print(name); }`)
	stmt := requireType[*ast.ForOfStatement](t, program.Statements[0])
	pattern := requireType[*ast.ArrayBindingPattern](t, stmt.Pattern)
	if len(pattern.Elements) != 2 {
		t.Fatalf("expected two array binding elements, got %d", len(pattern.Elements))
	}
}

func TestParserForOfObjectBindingPattern(t *testing.T) {
	program := parseProgram(t, `for (const { name } of entries) { print(name); }`)
	stmt := requireType[*ast.ForOfStatement](t, program.Statements[0])
	pattern := requireType[*ast.ObjectBindingPattern](t, stmt.Pattern)
	if len(pattern.Properties) != 1 {
		t.Fatalf("expected one object binding property, got %d", len(pattern.Properties))
	}
}

func TestParserRejectsForOfBindingInitializer(t *testing.T) {
	_, err := parseProgramError(`for (const item = 1 of items) { print(item); }`)
	if err == nil {
		t.Fatalf("expected for...of binding initializer error")
	}
}

func TestParserAllowsForOfDestructuringDefault(t *testing.T) {
	program := parseProgram(t, `for (const { name = "anon" } of entries) { print(name); }`)
	stmt := requireType[*ast.ForOfStatement](t, program.Statements[0])
	pattern := requireType[*ast.ObjectBindingPattern](t, stmt.Pattern)
	property := pattern.Properties[0]
	requireType[*ast.BindingDefault](t, property.Pattern)
}
