package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserNamedImportDeclaration(t *testing.T) {
	program := parseProgram(t, `import { add, twice as double } from "./math.js";`)
	decl := requireType[*ast.ImportDecl](t, program.Statements[0])
	if decl.Source != "./math.js" {
		t.Fatalf("expected import source ./math.js, got %q", decl.Source)
	}
	if len(decl.Specifiers) != 2 {
		t.Fatalf("expected two specifiers, got %d", len(decl.Specifiers))
	}
	if decl.Specifiers[1].Imported != "twice" || decl.Specifiers[1].Local != "double" {
		t.Fatalf("unexpected aliased import: %#v", decl.Specifiers[1])
	}
}

func TestParserDefaultNamedImportSpecifier(t *testing.T) {
	program := parseProgram(t, `import { default as main } from "./main.js";`)
	decl := requireType[*ast.ImportDecl](t, program.Statements[0])
	if len(decl.Specifiers) != 1 {
		t.Fatalf("expected one specifier, got %d", len(decl.Specifiers))
	}
	specifier := decl.Specifiers[0]
	if specifier.Imported != "default" || specifier.Local != "main" {
		t.Fatalf("unexpected default import specifier: %#v", specifier)
	}
}

func TestParserStringNamedImportSpecifier(t *testing.T) {
	program := parseProgram(t, `import { "kebab-name" as kebabName } from "./main.js";`)
	decl := requireType[*ast.ImportDecl](t, program.Statements[0])
	if len(decl.Specifiers) != 1 {
		t.Fatalf("expected one specifier, got %d", len(decl.Specifiers))
	}
	specifier := decl.Specifiers[0]
	if specifier.Imported != "kebab-name" || specifier.Local != "kebabName" {
		t.Fatalf("unexpected string import specifier: %#v", specifier)
	}
}

func TestParserKeywordNamedImportSpecifier(t *testing.T) {
	program := parseProgram(t, `import { class as className } from "./main.js";`)
	decl := requireType[*ast.ImportDecl](t, program.Statements[0])
	if len(decl.Specifiers) != 1 {
		t.Fatalf("expected one specifier, got %d", len(decl.Specifiers))
	}
	specifier := decl.Specifiers[0]
	if specifier.Imported != "class" || specifier.Local != "className" {
		t.Fatalf("unexpected keyword import specifier: %#v", specifier)
	}
}

func TestParserRejectsKeywordImportSpecifierWithoutAlias(t *testing.T) {
	_, err := parseProgramError(`import { class } from "./main.js";`)
	if err == nil {
		t.Fatalf("expected keyword import without alias to fail")
	}
}

func TestParserSideEffectImportDeclaration(t *testing.T) {
	program := parseProgram(t, `import "./setup.js";`)
	decl := requireType[*ast.ImportDecl](t, program.Statements[0])
	if !decl.SideEffect || decl.Source != "./setup.js" {
		t.Fatalf("unexpected side-effect import: %#v", decl)
	}
}

func TestParserExportedDeclaration(t *testing.T) {
	program := parseProgram(t, `export function add(a, b) { return a + b; }`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	fn := requireType[*ast.FunctionDecl](t, exportDecl.Declaration)
	if fn.Name != "add" {
		t.Fatalf("expected exported function add, got %q", fn.Name)
	}
}

func TestParserNamedExportList(t *testing.T) {
	program := parseProgram(t, `export { add, double as twice };`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if len(exportDecl.Specifiers) != 2 {
		t.Fatalf("expected two export specifiers, got %d", len(exportDecl.Specifiers))
	}
	if exportDecl.Specifiers[1].Local != "double" || exportDecl.Specifiers[1].Exported != "twice" {
		t.Fatalf("unexpected aliased export: %#v", exportDecl.Specifiers[1])
	}
}

func TestParserDefaultNamedExportSpecifier(t *testing.T) {
	program := parseProgram(t, `export { add as default };`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if len(exportDecl.Specifiers) != 1 {
		t.Fatalf("expected one export specifier, got %d", len(exportDecl.Specifiers))
	}
	specifier := exportDecl.Specifiers[0]
	if specifier.Local != "add" || specifier.Exported != "default" {
		t.Fatalf("unexpected default export specifier: %#v", specifier)
	}
}

func TestParserStringNamedExportSpecifier(t *testing.T) {
	program := parseProgram(t, `export { add as "kebab-name" };`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if len(exportDecl.Specifiers) != 1 {
		t.Fatalf("expected one export specifier, got %d", len(exportDecl.Specifiers))
	}
	specifier := exportDecl.Specifiers[0]
	if specifier.Local != "add" || specifier.Exported != "kebab-name" {
		t.Fatalf("unexpected string export specifier: %#v", specifier)
	}
}

func TestParserKeywordNamedExportSpecifier(t *testing.T) {
	program := parseProgram(t, `export { add as class };`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if len(exportDecl.Specifiers) != 1 {
		t.Fatalf("expected one export specifier, got %d", len(exportDecl.Specifiers))
	}
	specifier := exportDecl.Specifiers[0]
	if specifier.Local != "add" || specifier.Exported != "class" {
		t.Fatalf("unexpected keyword export specifier: %#v", specifier)
	}
}
