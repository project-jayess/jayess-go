package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
)

func TestParserRejectsUnsupportedAccessorClassField(t *testing.T) {
	_, err := parseProgramError(`class Widget { accessor value = 1; }`)
	if err == nil {
		t.Fatalf("expected accessor class field error")
	}
	if !strings.Contains(err.Error(), "accessor modifiers are not supported") {
		t.Fatalf("expected unsupported accessor diagnostic, got %v", err)
	}
}

func TestParserRejectsUnsupportedStaticAccessorClassField(t *testing.T) {
	_, err := parseProgramError(`class Widget { static accessor value = 1; }`)
	if err == nil {
		t.Fatalf("expected static accessor class field error")
	}
	if !strings.Contains(err.Error(), "accessor modifiers are not supported") {
		t.Fatalf("expected unsupported accessor diagnostic, got %v", err)
	}
}

func TestParserRejectsUnsupportedAccessorPrivateClassField(t *testing.T) {
	_, err := parseProgramError(`class Widget { accessor #value = 1; }`)
	if err == nil {
		t.Fatalf("expected accessor private class field error")
	}
	if !strings.Contains(err.Error(), "accessor modifiers are not supported") {
		t.Fatalf("expected unsupported accessor diagnostic, got %v", err)
	}
}

func TestParserStillAllowsAccessorClassMemberName(t *testing.T) {
	program := parseProgram(t, `
		class Widget {
			accessor() {
				return 1;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	method := classDecl.Members[0]
	if method.Name != "accessor" || method.Field {
		t.Fatalf("expected method named accessor, got %#v", method)
	}
}
