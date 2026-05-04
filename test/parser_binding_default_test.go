package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserArrayDestructuringDefault(t *testing.T) {
	program := parseProgram(t, `const [first = fallback] = values;`)
	decl := requireType[*ast.VariableDecl](t, program.Statements[0])
	array := requireType[*ast.ArrayBindingPattern](t, decl.Pattern)
	defaulted := requireType[*ast.BindingDefault](t, array.Elements[0])
	name := requireType[*ast.BindingName](t, defaulted.Pattern)
	if name.Name != "first" {
		t.Fatalf("expected binding name first, got %q", name.Name)
	}
	requireType[*ast.Identifier](t, defaulted.Value)
}

func TestParserObjectDestructuringDefault(t *testing.T) {
	program := parseProgram(t, `const { name = fallback, count: total = 1 } = item;`)
	decl := requireType[*ast.VariableDecl](t, program.Statements[0])
	object := requireType[*ast.ObjectBindingPattern](t, decl.Pattern)
	if len(object.Properties) != 2 {
		t.Fatalf("expected two properties, got %d", len(object.Properties))
	}
	requireType[*ast.BindingDefault](t, object.Properties[0].Pattern)
	requireType[*ast.BindingDefault](t, object.Properties[1].Pattern)
}

func TestParserKeepsIdentifierDeclarationInitializer(t *testing.T) {
	program := parseProgram(t, `const value = 1;`)
	decl := requireType[*ast.VariableDecl](t, program.Statements[0])
	requireType[*ast.BindingName](t, decl.Pattern)
	requireType[*ast.NumberLiteral](t, decl.Value)
}
