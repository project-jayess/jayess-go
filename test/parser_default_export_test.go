package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserDefaultExportExpression(t *testing.T) {
	program := parseProgram(t, `export default value;`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if !exportDecl.Default {
		t.Fatalf("expected default export")
	}
	ident := requireType[*ast.Identifier](t, exportDecl.Value)
	if ident.Name != "value" {
		t.Fatalf("expected exported value identifier, got %q", ident.Name)
	}
}

func TestParserDefaultExportFunctionDeclaration(t *testing.T) {
	program := parseProgram(t, `export default function main() { return 0; }`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if !exportDecl.Default {
		t.Fatalf("expected default export")
	}
	fn := requireType[*ast.FunctionDecl](t, exportDecl.Declaration)
	if fn.Name != "main" {
		t.Fatalf("expected exported function main, got %q", fn.Name)
	}
}

func TestParserDefaultExportAnonymousFunction(t *testing.T) {
	program := parseProgram(t, `export default function() { return 0; }`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if !exportDecl.Default {
		t.Fatalf("expected default export")
	}
	fn := requireType[*ast.FunctionExpression](t, exportDecl.Value)
	if fn.Name != "" {
		t.Fatalf("expected anonymous exported function, got %q", fn.Name)
	}
}

func TestParserDefaultExportAsyncFunctionDeclaration(t *testing.T) {
	program := parseProgram(t, `export default async function main() { return 0; }`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if !exportDecl.Default {
		t.Fatalf("expected default export")
	}
	fn := requireType[*ast.FunctionDecl](t, exportDecl.Declaration)
	if fn.Name != "main" || !fn.IsAsync {
		t.Fatalf("expected async exported function main, got %#v", fn)
	}
}

func TestParserDefaultExportAnonymousAsyncFunction(t *testing.T) {
	program := parseProgram(t, `export default async function() { return 0; }`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if !exportDecl.Default {
		t.Fatalf("expected default export")
	}
	fn := requireType[*ast.FunctionExpression](t, exportDecl.Value)
	if fn.Name != "" || !fn.IsAsync {
		t.Fatalf("expected anonymous async exported function, got %#v", fn)
	}
}

func TestParserDefaultExportAsyncGeneratorFunctionDeclaration(t *testing.T) {
	program := parseProgram(t, `export default async function* ids() { yield await next(); }`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if !exportDecl.Default {
		t.Fatalf("expected default export")
	}
	fn := requireType[*ast.FunctionDecl](t, exportDecl.Declaration)
	if fn.Name != "ids" || !fn.IsAsync || !fn.IsGenerator {
		t.Fatalf("expected async generator exported function ids, got %#v", fn)
	}
}

func TestParserDefaultExportAnonymousAsyncGeneratorFunction(t *testing.T) {
	program := parseProgram(t, `export default async function*() { yield await next(); }`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if !exportDecl.Default {
		t.Fatalf("expected default export")
	}
	fn := requireType[*ast.FunctionExpression](t, exportDecl.Value)
	if fn.Name != "" || !fn.IsAsync || !fn.IsGenerator {
		t.Fatalf("expected anonymous async generator exported function, got %#v", fn)
	}
}

func TestParserDefaultExportClassDeclaration(t *testing.T) {
	program := parseProgram(t, `export default class Counter {}`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if !exportDecl.Default {
		t.Fatalf("expected default export")
	}
	classDecl := requireType[*ast.ClassDecl](t, exportDecl.Declaration)
	if classDecl.Name != "Counter" {
		t.Fatalf("expected exported class Counter, got %q", classDecl.Name)
	}
}

func TestParserDefaultExportAnonymousClass(t *testing.T) {
	program := parseProgram(t, `export default class {}`)
	exportDecl := requireType[*ast.ExportDecl](t, program.Statements[0])
	if !exportDecl.Default {
		t.Fatalf("expected default export")
	}
	classDecl := requireType[*ast.ClassDecl](t, exportDecl.Declaration)
	if classDecl.Name != "" {
		t.Fatalf("expected anonymous exported class, got %q", classDecl.Name)
	}
}
