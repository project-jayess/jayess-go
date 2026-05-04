package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserObjectRestDestructuringDeclaration(t *testing.T) {
	program := parseProgram(t, `const { name, ...rest } = item;`)
	decl := requireType[*ast.VariableDecl](t, program.Statements[0])
	pattern := requireType[*ast.ObjectBindingPattern](t, decl.Pattern)
	if len(pattern.Properties) != 2 {
		t.Fatalf("expected two object binding properties, got %d", len(pattern.Properties))
	}
	restProperty := pattern.Properties[1]
	if !restProperty.Rest {
		t.Fatalf("expected rest property: %#v", restProperty)
	}
	rest := requireType[*ast.BindingRest](t, restProperty.Pattern)
	name := requireType[*ast.BindingName](t, rest.Pattern)
	if name.Name != "rest" {
		t.Fatalf("expected rest binding name rest, got %q", name.Name)
	}
}

func TestParserRejectsNonFinalObjectRestDestructuring(t *testing.T) {
	_, err := parseProgramError(`const { ...rest, name } = item;`)
	if err == nil {
		t.Fatalf("expected non-final object rest binding error")
	}
}
