package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserClassDeclarationWithConstructorAndMethod(t *testing.T) {
	program := parseProgram(t, `
		class Counter {
			constructor(value) {
				this.value = value;
			}
			total() {
				return this.value;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	if classDecl.Name != "Counter" {
		t.Fatalf("expected class Counter, got %q", classDecl.Name)
	}
	if len(classDecl.Members) != 2 {
		t.Fatalf("expected two class members, got %d", len(classDecl.Members))
	}
	if !classDecl.Members[0].Constructor {
		t.Fatalf("expected first member to be constructor")
	}
	if classDecl.Members[1].Name != "total" {
		t.Fatalf("expected method total, got %q", classDecl.Members[1].Name)
	}
}

func TestParserClassStaticMethod(t *testing.T) {
	program := parseProgram(t, `
		class Tools {
			static make(value) {
				return value;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	if len(classDecl.Members) != 1 {
		t.Fatalf("expected one class member, got %d", len(classDecl.Members))
	}
	if !classDecl.Members[0].Static {
		t.Fatalf("expected static class member")
	}
}

func TestParserClassAllowsEmptyElements(t *testing.T) {
	program := parseProgram(t, `
		class Tools {
			;
			value() {
				return 1;
			}
			;
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	if len(classDecl.Members) != 1 {
		t.Fatalf("expected one class member, got %d", len(classDecl.Members))
	}
	if classDecl.Members[0].Name != "value" {
		t.Fatalf("expected value method, got %#v", classDecl.Members[0])
	}
}

func TestParserClassMethodNamedStatic(t *testing.T) {
	program := parseProgram(t, `
		class Tools {
			static() {
				return 1;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	member := classDecl.Members[0]
	if member.Name != "static" || member.Static || member.Field {
		t.Fatalf("expected instance method named static, got %#v", member)
	}
}

func TestParserClassFieldNamedStatic(t *testing.T) {
	program := parseProgram(t, `
		class Tools {
			static = 1;
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	member := classDecl.Members[0]
	if member.Name != "static" || member.Static || !member.Field {
		t.Fatalf("expected instance field named static, got %#v", member)
	}
}

func TestParserClassKeywordMethod(t *testing.T) {
	program := parseProgram(t, `
		class Tools {
			default(value) {
				return value;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	member := classDecl.Members[0]
	if member.Name != "default" || member.Field {
		t.Fatalf("expected default method, got %#v", member)
	}
}
