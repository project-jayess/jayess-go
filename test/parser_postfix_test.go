package test

import (
	"testing"

	"jayess-go/ast"
	"jayess-go/lexer"
	"jayess-go/parser"
)

func TestParserExpressionMemberCallAndIndex(t *testing.T) {
	expr := parseExpression(t, "console.log(items[0], label)")
	call := requireType[*ast.InvokeExpression](t, expr)
	if len(call.Arguments) != 2 {
		t.Fatalf("expected 2 call arguments, got %d", len(call.Arguments))
	}
	member := requireType[*ast.MemberExpression](t, call.Callee)
	if member.Property != "log" {
		t.Fatalf("expected log member, got %q", member.Property)
	}
	index := requireType[*ast.IndexExpression](t, call.Arguments[0])
	if ident := requireType[*ast.Identifier](t, index.Target); ident.Name != "items" {
		t.Fatalf("expected items index target, got %q", ident.Name)
	}
}

func TestParserExpressionOptionalChaining(t *testing.T) {
	expr := parseExpression(t, "user?.profile?.getName?.()")
	call := requireType[*ast.InvokeExpression](t, expr)
	if !call.Optional {
		t.Fatalf("expected optional call")
	}
	getName := requireType[*ast.MemberExpression](t, call.Callee)
	if !getName.Optional || getName.Property != "getName" {
		t.Fatalf("expected optional getName member, got %#v", getName)
	}
	profile := requireType[*ast.MemberExpression](t, getName.Target)
	if !profile.Optional || profile.Property != "profile" {
		t.Fatalf("expected optional profile member, got %#v", profile)
	}
}

func TestParserProgramAssignsMemberAndIndexTargets(t *testing.T) {
	program := parseProgram(t, "user.name = \"Jay\";\nitems[0] += 1;")
	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}
	memberAssign := requireType[*ast.AssignmentStatement](t, program.Statements[0])
	requireType[*ast.MemberExpression](t, memberAssign.Target)
	indexAssign := requireType[*ast.AssignmentStatement](t, program.Statements[1])
	requireType[*ast.IndexExpression](t, indexAssign.Target)
}

func TestParserRejectsOptionalChainAssignmentTarget(t *testing.T) {
	_, err := parser.New(lexer.New(`user?.name = "Jay";`)).ParseProgram()
	if err == nil {
		t.Fatalf("expected optional chain assignment target error")
	}
}

func TestParserRejectsNestedOptionalChainAssignmentTarget(t *testing.T) {
	_, err := parser.New(lexer.New(`user?.profile.name = "Jay";`)).ParseProgram()
	if err == nil {
		t.Fatalf("expected nested optional chain assignment target error")
	}
}

func TestParserRejectsOptionalChainUpdateTarget(t *testing.T) {
	_, err := parser.New(lexer.New(`user?.count++;`)).ParseProgram()
	if err == nil {
		t.Fatalf("expected optional chain update target error")
	}
}
