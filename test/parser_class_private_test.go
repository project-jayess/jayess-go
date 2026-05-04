package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserClassPrivateFieldAndMethod(t *testing.T) {
	program := parseProgram(t, `
		class Counter {
			#value = 1;
			#next() {
				return this.#value;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	field := classDecl.Members[0]
	if !field.Field || !field.Private || field.Name != "value" {
		t.Fatalf("expected private value field, got %#v", field)
	}
	method := classDecl.Members[1]
	if method.Field || !method.Private || method.Name != "next" {
		t.Fatalf("expected private next method, got %#v", method)
	}
	ret := requireType[*ast.ReturnStatement](t, method.Body[0])
	member := requireType[*ast.MemberExpression](t, ret.Value)
	if !member.Private || member.Property != "value" {
		t.Fatalf("expected private value access, got %#v", member)
	}
}

func TestParserClassPrivateAccessors(t *testing.T) {
	program := parseProgram(t, `
		class Counter {
			get #value() {
				return this.#count;
			}
			set #value(next) {
				this.#count = next;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	if !classDecl.Members[0].Getter || !classDecl.Members[0].Private {
		t.Fatalf("expected private getter, got %#v", classDecl.Members[0])
	}
	if !classDecl.Members[1].Setter || !classDecl.Members[1].Private {
		t.Fatalf("expected private setter, got %#v", classDecl.Members[1])
	}
}

func TestParserClassStaticPrivateField(t *testing.T) {
	program := parseProgram(t, `
		class Config {
			static #version = 1;
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	field := classDecl.Members[0]
	if !field.Static || !field.Private || !field.Field {
		t.Fatalf("expected static private field, got %#v", field)
	}
}

func TestParserClassPrivateAsyncMethod(t *testing.T) {
	program := parseProgram(t, `
		class Loader {
			async #load(read) {
				return await read();
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	method := classDecl.Members[0]
	if method.Name != "load" || !method.Private || !method.IsAsync || method.IsGenerator {
		t.Fatalf("unexpected private async method: %#v", method)
	}
}

func TestParserClassPrivateGeneratorMethod(t *testing.T) {
	program := parseProgram(t, `
		class Ids {
			*#values(value) {
				yield value;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	method := classDecl.Members[0]
	if method.Name != "values" || !method.Private || !method.IsGenerator || method.IsAsync {
		t.Fatalf("unexpected private generator method: %#v", method)
	}
}
