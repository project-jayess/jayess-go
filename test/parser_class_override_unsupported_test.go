package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
)

func TestParserRejectsUnsupportedOverrideClassMethod(t *testing.T) {
	_, err := parseProgramError(`class Widget { override render() { return 1; } }`)
	if err == nil {
		t.Fatalf("expected override class method error")
	}
	if !strings.Contains(err.Error(), "override modifiers are not supported") {
		t.Fatalf("expected unsupported override diagnostic, got %v", err)
	}
}

func TestParserRejectsUnsupportedStaticOverrideClassMethod(t *testing.T) {
	_, err := parseProgramError(`class Widget { static override render() { return 1; } }`)
	if err == nil {
		t.Fatalf("expected static override class method error")
	}
	if !strings.Contains(err.Error(), "override modifiers are not supported") {
		t.Fatalf("expected unsupported override diagnostic, got %v", err)
	}
}

func TestParserRejectsUnsupportedOverridePrivateClassMethod(t *testing.T) {
	_, err := parseProgramError(`class Widget { override #render() { return 1; } }`)
	if err == nil {
		t.Fatalf("expected override private class method error")
	}
	if !strings.Contains(err.Error(), "override modifiers are not supported") {
		t.Fatalf("expected unsupported override diagnostic, got %v", err)
	}
}

func TestParserStillAllowsOverrideClassMemberName(t *testing.T) {
	program := parseProgram(t, `
		class Widget {
			override() {
				return 1;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	method := classDecl.Members[0]
	if method.Name != "override" || method.Field {
		t.Fatalf("expected method named override, got %#v", method)
	}
}
