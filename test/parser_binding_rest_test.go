package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
)

func TestParserArrayDestructuringRest(t *testing.T) {
	program := parseProgram(t, `const [first, ...rest] = values;`)
	decl := requireType[*ast.VariableDecl](t, program.Statements[0])
	array := requireType[*ast.ArrayBindingPattern](t, decl.Pattern)
	if len(array.Elements) != 2 {
		t.Fatalf("expected two binding elements, got %d", len(array.Elements))
	}
	rest := requireType[*ast.BindingRest](t, array.Elements[1])
	name := requireType[*ast.BindingName](t, rest.Pattern)
	if name.Name != "rest" {
		t.Fatalf("expected rest binding name rest, got %q", name.Name)
	}
}

func TestParserRejectsNonFinalArrayRestBinding(t *testing.T) {
	_, err := parseProgramError(`const [...rest, last] = values;`)
	if err == nil {
		t.Fatalf("expected non-final rest binding error")
	}
	if !strings.Contains(err.Error(), "rest binding must be last") {
		t.Fatalf("expected clear non-final rest diagnostic, got %v", err)
	}
}

func TestParserRejectsArrayRestBindingDefault(t *testing.T) {
	_, err := parseProgramError(`const [...rest = fallback] = values;`)
	if err == nil {
		t.Fatalf("expected rest binding default error")
	}
	if !strings.Contains(err.Error(), "rest binding cannot have a default value") {
		t.Fatalf("expected clear rest default diagnostic, got %v", err)
	}
}

func TestParserRejectsMissingArrayRestBindingTarget(t *testing.T) {
	_, err := parseProgramError(`const [...] = values;`)
	if err == nil {
		t.Fatalf("expected missing array rest target error")
	}
	if !strings.Contains(err.Error(), "rest binding requires a target") {
		t.Fatalf("expected clear missing rest target diagnostic, got %v", err)
	}
}

func TestParserRejectsNonFinalObjectRestBinding(t *testing.T) {
	_, err := parseProgramError(`const { ...rest, last } = value;`)
	if err == nil {
		t.Fatalf("expected non-final object rest binding error")
	}
	if !strings.Contains(err.Error(), "rest binding must be last") {
		t.Fatalf("expected clear non-final rest diagnostic, got %v", err)
	}
}

func TestParserRejectsObjectRestBindingDefault(t *testing.T) {
	_, err := parseProgramError(`const { ...rest = fallback } = value;`)
	if err == nil {
		t.Fatalf("expected object rest binding default error")
	}
	if !strings.Contains(err.Error(), "rest binding cannot have a default value") {
		t.Fatalf("expected clear rest default diagnostic, got %v", err)
	}
}

func TestParserRejectsMissingObjectRestBindingTarget(t *testing.T) {
	_, err := parseProgramError(`const { ... } = value;`)
	if err == nil {
		t.Fatalf("expected missing object rest target error")
	}
	if !strings.Contains(err.Error(), "rest binding requires a target") {
		t.Fatalf("expected clear missing rest target diagnostic, got %v", err)
	}
}
