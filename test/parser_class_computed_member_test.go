package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserClassComputedMethod(t *testing.T) {
	program := parseProgram(t, `
		class Tools {
			[methodName](value) {
				return value;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	member := classDecl.Members[0]
	if !member.Computed || member.Field {
		t.Fatalf("expected computed method, got %#v", member)
	}
	requireType[*ast.Identifier](t, member.KeyExpr)
}

func TestParserClassComputedField(t *testing.T) {
	program := parseProgram(t, `
		class Config {
			[fieldName] = 1;
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	member := classDecl.Members[0]
	if !member.Computed || !member.Field {
		t.Fatalf("expected computed field, got %#v", member)
	}
	requireType[*ast.Identifier](t, member.KeyExpr)
	requireType[*ast.NumberLiteral](t, member.Value)
}

func TestParserClassComputedAccessor(t *testing.T) {
	program := parseProgram(t, `
		class Config {
			get [fieldName]() {
				return 1;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	member := classDecl.Members[0]
	if !member.Computed || !member.Getter {
		t.Fatalf("expected computed getter, got %#v", member)
	}
	requireType[*ast.Identifier](t, member.KeyExpr)
}
