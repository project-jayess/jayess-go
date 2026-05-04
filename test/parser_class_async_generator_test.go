package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserClassAsyncMethod(t *testing.T) {
	program := parseProgram(t, `
		class Loader {
			async load(read) {
				return await read();
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	member := classDecl.Members[0]
	if member.Name != "load" || !member.IsAsync || member.IsGenerator {
		t.Fatalf("unexpected async class method: %#v", member)
	}
}

func TestParserClassGeneratorMethod(t *testing.T) {
	program := parseProgram(t, `
		class Ids {
			*values(value) {
				yield value;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	member := classDecl.Members[0]
	if member.Name != "values" || !member.IsGenerator || member.IsAsync {
		t.Fatalf("unexpected generator class method: %#v", member)
	}
}

func TestParserClassAsyncGeneratorComputedMethod(t *testing.T) {
	program := parseProgram(t, `
		class Ids {
			async *[method](value) {
				yield await value;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	member := classDecl.Members[0]
	if !member.Computed || !member.IsAsync || !member.IsGenerator {
		t.Fatalf("unexpected async generator computed class method: %#v", member)
	}
}

func TestParserClassMethodNamedAsync(t *testing.T) {
	program := parseProgram(t, `
		class Loader {
			async() {
				return 1;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	member := classDecl.Members[0]
	if member.Name != "async" || member.IsAsync {
		t.Fatalf("expected method named async, got %#v", member)
	}
}

func TestParserClassAsyncMethodLineBreakStartsField(t *testing.T) {
	program := parseProgram(t, `
		class Loader {
			async
			load() {}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	if len(classDecl.Members) != 2 {
		t.Fatalf("expected async field and load method, got %d members", len(classDecl.Members))
	}
	if !classDecl.Members[0].Field || classDecl.Members[0].Name != "async" {
		t.Fatalf("expected async class field, got %#v", classDecl.Members[0])
	}
	if classDecl.Members[1].Name != "load" || classDecl.Members[1].IsAsync {
		t.Fatalf("expected non-async load method, got %#v", classDecl.Members[1])
	}
}
