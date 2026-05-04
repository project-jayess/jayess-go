package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
)

func TestParserRejectsUnsupportedReadonlyClassField(t *testing.T) {
	_, err := parseProgramError(`class Widget { readonly value = 1; }`)
	if err == nil {
		t.Fatalf("expected readonly class field error")
	}
	if !strings.Contains(err.Error(), "readonly modifiers are not supported") {
		t.Fatalf("expected unsupported readonly diagnostic, got %v", err)
	}
}

func TestParserRejectsUnsupportedStaticReadonlyClassField(t *testing.T) {
	_, err := parseProgramError(`class Widget { static readonly value = 1; }`)
	if err == nil {
		t.Fatalf("expected static readonly class field error")
	}
	if !strings.Contains(err.Error(), "readonly modifiers are not supported") {
		t.Fatalf("expected unsupported readonly diagnostic, got %v", err)
	}
}

func TestParserRejectsUnsupportedReadonlyPrivateClassField(t *testing.T) {
	_, err := parseProgramError(`class Widget { readonly #value = 1; }`)
	if err == nil {
		t.Fatalf("expected readonly private class field error")
	}
	if !strings.Contains(err.Error(), "readonly modifiers are not supported") {
		t.Fatalf("expected unsupported readonly diagnostic, got %v", err)
	}
}

func TestParserStillAllowsReadonlyClassMemberName(t *testing.T) {
	program := parseProgram(t, `
		class Widget {
			readonly = 1;
			readonlyMethod() {
				return this.readonly;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	field := classDecl.Members[0]
	if field.Name != "readonly" || !field.Field {
		t.Fatalf("expected field named readonly, got %#v", field)
	}
}
