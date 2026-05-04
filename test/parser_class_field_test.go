package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserClassInstanceFields(t *testing.T) {
	program := parseProgram(t, `
		class Counter {
			value = 1;
			name;
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	if len(classDecl.Members) != 2 {
		t.Fatalf("expected two class members, got %d", len(classDecl.Members))
	}
	first := classDecl.Members[0]
	if !first.Field || first.Name != "value" {
		t.Fatalf("expected value field, got %#v", first)
	}
	requireType[*ast.NumberLiteral](t, first.Value)
	second := classDecl.Members[1]
	if !second.Field || second.Name != "name" || second.Value != nil {
		t.Fatalf("expected empty name field, got %#v", second)
	}
}

func TestParserClassStaticField(t *testing.T) {
	program := parseProgram(t, `
		class Config {
			static version = 1;
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	field := classDecl.Members[0]
	if !field.Field || !field.Static || field.Name != "version" {
		t.Fatalf("expected static version field, got %#v", field)
	}
}

func TestParserClassKeywordField(t *testing.T) {
	program := parseProgram(t, `
		class Config {
			default = 1;
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	field := classDecl.Members[0]
	if !field.Field || field.Name != "default" {
		t.Fatalf("expected default field, got %#v", field)
	}
}
