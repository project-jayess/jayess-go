package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserClassExtends(t *testing.T) {
	program := parseProgram(t, `
		class Counter extends BaseCounter {
			total() {
				return super.total();
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	if classDecl.Name != "Counter" {
		t.Fatalf("expected class Counter, got %q", classDecl.Name)
	}
	base := requireType[*ast.Identifier](t, classDecl.SuperClass)
	if base.Name != "BaseCounter" {
		t.Fatalf("expected base BaseCounter, got %q", base.Name)
	}
	method := classDecl.Members[0]
	ret := requireType[*ast.ReturnStatement](t, method.Body[0])
	call := requireType[*ast.InvokeExpression](t, ret.Value)
	member := requireType[*ast.MemberExpression](t, call.Callee)
	requireType[*ast.SuperExpression](t, member.Target)
}

func TestParserSuperConstructorCall(t *testing.T) {
	program := parseProgram(t, `
		class Counter extends BaseCounter {
			constructor(value) {
				super(value);
			}
		}
	`)
	classDecl := requireType[*ast.ClassDecl](t, program.Statements[0])
	stmt := requireType[*ast.ExpressionStatement](t, classDecl.Members[0].Body[0])
	call := requireType[*ast.InvokeExpression](t, stmt.Expression)
	requireType[*ast.SuperExpression](t, call.Callee)
	if len(call.Arguments) != 1 {
		t.Fatalf("expected one super argument, got %d", len(call.Arguments))
	}
}

func TestParserClassMethodOverrideShape(t *testing.T) {
	program := parseProgram(t, `
		class BaseCounter {
			total() {
				return 1;
			}
		}
		class Counter extends BaseCounter {
			total() {
				return super.total() + 1;
			}
		}
	`)
	derived := requireType[*ast.ClassDecl](t, program.Statements[1])
	method := derived.Members[0]
	if method.Name != "total" {
		t.Fatalf("expected overriding total method, got %q", method.Name)
	}
	ret := requireType[*ast.ReturnStatement](t, method.Body[0])
	add := requireType[*ast.BinaryExpression](t, ret.Value)
	call := requireType[*ast.InvokeExpression](t, add.Left)
	member := requireType[*ast.MemberExpression](t, call.Callee)
	requireType[*ast.SuperExpression](t, member.Target)
}

func TestParserPrototypeChainMethodCallShape(t *testing.T) {
	program := parseProgram(t, `
		class BaseCounter {
			total() {
				return 1;
			}
		}
		class Counter extends BaseCounter {
			total() {
				return BaseCounter.prototype.total.call(this) + 1;
			}
		}
	`)
	derived := requireType[*ast.ClassDecl](t, program.Statements[1])
	ret := requireType[*ast.ReturnStatement](t, derived.Members[0].Body[0])
	add := requireType[*ast.BinaryExpression](t, ret.Value)
	call := requireType[*ast.InvokeExpression](t, add.Left)
	callMember := requireType[*ast.MemberExpression](t, call.Callee)
	if callMember.Property != "call" {
		t.Fatalf("expected call helper, got %q", callMember.Property)
	}
	totalMember := requireType[*ast.MemberExpression](t, callMember.Target)
	if totalMember.Property != "total" {
		t.Fatalf("expected prototype total member, got %q", totalMember.Property)
	}
}
