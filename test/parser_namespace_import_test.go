package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserNamespaceImportDeclaration(t *testing.T) {
	program := parseProgram(t, `import * as math from "./math.js";`)
	decl := requireType[*ast.ImportDecl](t, program.Statements[0])
	if decl.Source != "./math.js" {
		t.Fatalf("expected import source ./math.js, got %q", decl.Source)
	}
	if len(decl.Specifiers) != 1 {
		t.Fatalf("expected one specifier, got %d", len(decl.Specifiers))
	}
	specifier := decl.Specifiers[0]
	if !specifier.Namespace || specifier.Imported != "*" || specifier.Local != "math" {
		t.Fatalf("unexpected namespace import: %#v", specifier)
	}
}

func TestParserDefaultAndNamedImportDeclaration(t *testing.T) {
	program := parseProgram(t, `import main, { add as sum } from "./math.js";`)
	decl := requireType[*ast.ImportDecl](t, program.Statements[0])
	if len(decl.Specifiers) != 2 {
		t.Fatalf("expected two specifiers, got %d", len(decl.Specifiers))
	}
	if !decl.Specifiers[0].Default || decl.Specifiers[0].Local != "main" {
		t.Fatalf("unexpected default import: %#v", decl.Specifiers[0])
	}
	if decl.Specifiers[1].Imported != "add" || decl.Specifiers[1].Local != "sum" {
		t.Fatalf("unexpected named import: %#v", decl.Specifiers[1])
	}
}
