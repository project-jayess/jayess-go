package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
)

func TestParserRejectsUnsupportedModuleDeclaration(t *testing.T) {
	_, err := parseProgramError(`module App { export const value = 1; }`)
	requireModuleDeclarationError(t, err)
}

func TestParserRejectsUnsupportedStringModuleDeclaration(t *testing.T) {
	_, err := parseProgramError(`module "pkg" { export const value = 1; }`)
	requireModuleDeclarationError(t, err)
}

func TestParserStillAllowsModuleIdentifierExpression(t *testing.T) {
	program := parseProgram(t, `
		module;
		moduleName;
	`)
	if len(program.Statements) != 2 {
		t.Fatalf("expected two expression statements, got %d", len(program.Statements))
	}
	first := requireType[*ast.ExpressionStatement](t, program.Statements[0])
	ident := requireType[*ast.Identifier](t, first.Expression)
	if ident.Name != "module" {
		t.Fatalf("expected module identifier, got %q", ident.Name)
	}
}

func requireModuleDeclarationError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected module declaration error")
	}
	if !strings.Contains(err.Error(), "module declarations are not supported") {
		t.Fatalf("expected unsupported module declaration diagnostic, got %v", err)
	}
}
