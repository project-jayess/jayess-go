package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserClassGettersAndSetters(t *testing.T) {
	program := parseProgram(t, `
		class Counter {
			get value() {
				return this.count;
			}
			set value(next) {
				this.count = next;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	if len(classDecl.Members) != 2 {
		t.Fatalf("expected two accessors, got %d", len(classDecl.Members))
	}
	if !classDecl.Members[0].Getter || classDecl.Members[0].Name != "value" {
		t.Fatalf("expected value getter, got %#v", classDecl.Members[0])
	}
	if !classDecl.Members[1].Setter || len(classDecl.Members[1].Params) != 1 {
		t.Fatalf("expected value setter with one parameter, got %#v", classDecl.Members[1])
	}
}

func TestParserClassStaticGetter(t *testing.T) {
	program := parseProgram(t, `
		class Config {
			static get version() {
				return 1;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	member := classDecl.Members[0]
	if !member.Static || !member.Getter || member.Name != "version" {
		t.Fatalf("expected static version getter, got %#v", member)
	}
}

func TestParserClassKeywordGetter(t *testing.T) {
	program := parseProgram(t, `
		class Counter {
			get default() {
				return this.value;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	member := classDecl.Members[0]
	if !member.Getter || member.Name != "default" {
		t.Fatalf("expected default getter, got %#v", member)
	}
}

func TestParserRejectsGetterWithParameter(t *testing.T) {
	_, err := parseProgramError(`class Bad { get value(next) { return next; } }`)
	if err == nil {
		t.Fatalf("expected getter parameter error")
	}
}

func TestParserRejectsSetterWithoutParameter(t *testing.T) {
	_, err := parseProgramError(`class Bad { set value() {} }`)
	if err == nil {
		t.Fatalf("expected setter parameter error")
	}
}

func TestParserRejectsSetterRestParameter(t *testing.T) {
	_, err := parseProgramError(`class Bad { set value(...next) {} }`)
	if err == nil {
		t.Fatalf("expected setter rest parameter error")
	}
}

func TestParserRejectsComputedSetterRestParameter(t *testing.T) {
	_, err := parseProgramError(`class Bad { set [key](...next) {} }`)
	if err == nil {
		t.Fatalf("expected computed setter rest parameter error")
	}
}
