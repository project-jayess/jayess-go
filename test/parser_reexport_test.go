package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserNamedReExportList(t *testing.T) {
	program := parseProgram(t, `export { add as sum } from "./math.js";`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if exportDecl.Source != "./math.js" {
		t.Fatalf("expected re-export source ./math.js, got %q", exportDecl.Source)
	}
	if len(exportDecl.Specifiers) != 1 {
		t.Fatalf("expected one export specifier, got %d", len(exportDecl.Specifiers))
	}
	specifier := exportDecl.Specifiers[0]
	if specifier.Local != "add" || specifier.Exported != "sum" {
		t.Fatalf("unexpected re-export specifier: %#v", specifier)
	}
}

func TestParserDefaultNamedReExportList(t *testing.T) {
	program := parseProgram(t, `export { default as main } from "./main.js";`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if exportDecl.Source != "./main.js" {
		t.Fatalf("expected re-export source ./main.js, got %q", exportDecl.Source)
	}
	specifier := exportDecl.Specifiers[0]
	if specifier.Local != "default" || specifier.Exported != "main" {
		t.Fatalf("unexpected default re-export specifier: %#v", specifier)
	}
}

func TestParserKeywordNamedReExportList(t *testing.T) {
	program := parseProgram(t, `export { class as className } from "./main.js";`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if exportDecl.Source != "./main.js" {
		t.Fatalf("expected re-export source ./main.js, got %q", exportDecl.Source)
	}
	specifier := exportDecl.Specifiers[0]
	if specifier.Local != "class" || specifier.Exported != "className" {
		t.Fatalf("unexpected keyword re-export specifier: %#v", specifier)
	}
}

func TestParserExportAllDeclaration(t *testing.T) {
	program := parseProgram(t, `export * from "./more.js";`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if !exportDecl.All || exportDecl.Source != "./more.js" {
		t.Fatalf("unexpected export all declaration: %#v", exportDecl)
	}
}

func TestParserNamespaceReExportDeclaration(t *testing.T) {
	program := parseProgram(t, `export * as math from "./more.js";`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if exportDecl.Namespace != "math" || exportDecl.Source != "./more.js" {
		t.Fatalf("unexpected namespace re-export declaration: %#v", exportDecl)
	}
}

func TestParserStringNamespaceReExportDeclaration(t *testing.T) {
	program := parseProgram(t, `export * as "math-tools" from "./more.js";`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if exportDecl.Namespace != "math-tools" || exportDecl.Source != "./more.js" {
		t.Fatalf("unexpected string namespace re-export declaration: %#v", exportDecl)
	}
}

func TestParserDefaultNamespaceReExportDeclaration(t *testing.T) {
	program := parseProgram(t, `export * as default from "./more.js";`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if exportDecl.Namespace != "default" || exportDecl.Source != "./more.js" {
		t.Fatalf("unexpected default namespace re-export declaration: %#v", exportDecl)
	}
}
