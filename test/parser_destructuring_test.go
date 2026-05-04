package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserArrayDestructuringDeclaration(t *testing.T) {
	program := parseProgram(t, `const [first, second] = values;`)
	decl := requireType[*ast.VariableDecl](t, program.Statements[0])
	pattern := requireType[*ast.ArrayBindingPattern](t, decl.Pattern)
	if len(pattern.Elements) != 2 {
		t.Fatalf("expected two array binding elements, got %d", len(pattern.Elements))
	}
	first := requireType[*ast.BindingName](t, pattern.Elements[0])
	if first.Name != "first" {
		t.Fatalf("expected first binding name first, got %q", first.Name)
	}
}

func TestParserArrayDestructuringElisions(t *testing.T) {
	program := parseProgram(t, `const [first, , third] = values;`)
	decl := requireType[*ast.VariableDecl](t, program.Statements[0])
	pattern := requireType[*ast.ArrayBindingPattern](t, decl.Pattern)
	if len(pattern.Elements) != 3 {
		t.Fatalf("expected three array binding elements, got %d", len(pattern.Elements))
	}
	if pattern.Elements[1] != nil {
		t.Fatalf("expected skipped middle element, got %#v", pattern.Elements[1])
	}
	third := requireType[*ast.BindingName](t, pattern.Elements[2])
	if third.Name != "third" {
		t.Fatalf("expected third binding name third, got %q", third.Name)
	}
}

func TestParserObjectDestructuringDeclaration(t *testing.T) {
	program := parseProgram(t, `var { name, count: total } = item;`)
	decl := requireType[*ast.VariableDecl](t, program.Statements[0])
	pattern := requireType[*ast.ObjectBindingPattern](t, decl.Pattern)
	if len(pattern.Properties) != 2 {
		t.Fatalf("expected two object binding properties, got %d", len(pattern.Properties))
	}
	if pattern.Properties[0].Key != "name" {
		t.Fatalf("expected first key name, got %q", pattern.Properties[0].Key)
	}
	total := requireType[*ast.BindingName](t, pattern.Properties[1].Pattern)
	if total.Name != "total" {
		t.Fatalf("expected alias total, got %q", total.Name)
	}
}

func TestParserObjectDestructuringKeywordPropertyName(t *testing.T) {
	program := parseProgram(t, `const { default: value, class: kind } = item;`)
	decl := requireType[*ast.VariableDecl](t, program.Statements[0])
	pattern := requireType[*ast.ObjectBindingPattern](t, decl.Pattern)
	if len(pattern.Properties) != 2 {
		t.Fatalf("expected two object binding properties, got %d", len(pattern.Properties))
	}
	if pattern.Properties[0].Key != "default" || pattern.Properties[1].Key != "class" {
		t.Fatalf("unexpected keyword binding properties: %#v", pattern.Properties)
	}
	value := requireType[*ast.BindingName](t, pattern.Properties[0].Pattern)
	if value.Name != "value" {
		t.Fatalf("expected alias value, got %q", value.Name)
	}
}

func TestParserObjectDestructuringComputedPropertyName(t *testing.T) {
	program := parseProgram(t, `const { [key]: value } = item;`)
	decl := requireType[*ast.VariableDecl](t, program.Statements[0])
	pattern := requireType[*ast.ObjectBindingPattern](t, decl.Pattern)
	if len(pattern.Properties) != 1 {
		t.Fatalf("expected one object binding property, got %d", len(pattern.Properties))
	}
	property := pattern.Properties[0]
	if !property.Computed {
		t.Fatalf("expected computed binding property, got %#v", property)
	}
	requireType[*ast.Identifier](t, property.KeyExpr)
	value := requireType[*ast.BindingName](t, property.Pattern)
	if value.Name != "value" {
		t.Fatalf("expected alias value, got %q", value.Name)
	}
}

func TestParserRejectsKeywordObjectDestructuringShorthand(t *testing.T) {
	_, err := parseProgramError(`const { default } = item;`)
	if err == nil {
		t.Fatalf("expected keyword shorthand binding error")
	}
}

func TestParserRejectsDestructuringDeclarationWithoutInitializer(t *testing.T) {
	_, err := parseProgramError(`var [first];`)
	if err == nil {
		t.Fatalf("expected destructuring initializer error")
	}
}
