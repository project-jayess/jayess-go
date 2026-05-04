package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserClassStaticBlock(t *testing.T) {
	program := parseProgram(t, `
		class Counter {
			static {
				this.value = 1;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	if len(classDecl.Members) != 1 {
		t.Fatalf("expected one class member, got %d", len(classDecl.Members))
	}
	member := classDecl.Members[0]
	if !member.StaticBlock || !member.Static || len(member.Body) != 1 {
		t.Fatalf("unexpected static block member: %#v", member)
	}
}

func TestParserStaticBlockDoesNotConsumeStaticMethodName(t *testing.T) {
	program := parseProgram(t, `
		class Tools {
			static() {
				return 1;
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	member := classDecl.Members[0]
	if member.StaticBlock || member.Name != "static" {
		t.Fatalf("expected method named static, got %#v", member)
	}
}
