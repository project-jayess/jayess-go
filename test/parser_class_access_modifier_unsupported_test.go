package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
)

func TestParserRejectsUnsupportedPublicClassMember(t *testing.T) {
	_, err := parseProgramError(`class Widget { public value = 1; }`)
	requireUnsupportedClassAccessModifierError(t, err)
}

func TestParserRejectsUnsupportedPrivateClassMember(t *testing.T) {
	_, err := parseProgramError(`class Widget { private value = 1; }`)
	requireUnsupportedClassAccessModifierError(t, err)
}

func TestParserRejectsUnsupportedProtectedClassMember(t *testing.T) {
	_, err := parseProgramError(`class Widget { protected value = 1; }`)
	requireUnsupportedClassAccessModifierError(t, err)
}

func TestParserRejectsUnsupportedStaticProtectedClassMember(t *testing.T) {
	_, err := parseProgramError(`class Widget { static protected value = 1; }`)
	requireUnsupportedClassAccessModifierError(t, err)
}

func TestParserStillAllowsAccessModifierClassMemberNames(t *testing.T) {
	program := parseProgram(t, `
		class Widget {
			public() {
				return 1;
			}
			private = 2;
			protected() {
				return this.private;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	if classDecl.Members[0].Name != "public" || classDecl.Members[0].Field {
		t.Fatalf("expected method named public, got %#v", classDecl.Members[0])
	}
	if classDecl.Members[1].Name != "private" || !classDecl.Members[1].Field {
		t.Fatalf("expected field named private, got %#v", classDecl.Members[1])
	}
	if classDecl.Members[2].Name != "protected" || classDecl.Members[2].Field {
		t.Fatalf("expected method named protected, got %#v", classDecl.Members[2])
	}
}

func requireUnsupportedClassAccessModifierError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected class access modifier error")
	}
	if !strings.Contains(err.Error(), "class access modifiers are not supported") {
		t.Fatalf("expected unsupported class access modifier diagnostic, got %v", err)
	}
}
