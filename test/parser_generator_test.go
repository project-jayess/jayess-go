package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserGeneratorFunctionDeclaration(t *testing.T) {
	program := parseProgram(t, `function* ids(value) { yield value; }`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[0])
	if !fn.IsGenerator || fn.Name != "ids" {
		t.Fatalf("unexpected generator declaration: %#v", fn)
	}
	stmt := requireType[*ast.ExpressionStatement](t, fn.Body[0])
	requireType[*ast.YieldExpression](t, stmt.Expression)
}

func TestParserGeneratorFunctionExpression(t *testing.T) {
	expr := parseExpression(t, `function* ids(value) { yield value; }`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if !fn.IsGenerator || fn.Name != "ids" {
		t.Fatalf("unexpected generator expression: %#v", fn)
	}
}

func TestParserBareYieldExpression(t *testing.T) {
	program := parseProgram(t, `function* ids() { yield; }`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[0])
	stmt := requireType[*ast.ExpressionStatement](t, fn.Body[0])
	yield := requireType[*ast.YieldExpression](t, stmt.Expression)
	if yield.Value != nil {
		t.Fatalf("expected bare yield value to be nil, got %#v", yield.Value)
	}
}

func TestParserDelegateYieldExpression(t *testing.T) {
	program := parseProgram(t, `function* ids(values) { yield* values; }`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[0])
	stmt := requireType[*ast.ExpressionStatement](t, fn.Body[0])
	yield := requireType[*ast.YieldExpression](t, stmt.Expression)
	if !yield.Delegate {
		t.Fatalf("expected delegate yield")
	}
	requireType[*ast.Identifier](t, yield.Value)
}

func TestParserGeneratorNextInvocation(t *testing.T) {
	program := parseProgram(t, `
		function* ids() { yield 1; }
		const first = ids().next();
	`)
	decl := requireType[*ast.VariableDecl](t, program.Statements[1])
	invoke := requireType[*ast.InvokeExpression](t, decl.Value)
	member := requireType[*ast.MemberExpression](t, invoke.Callee)
	if member.Property != "next" {
		t.Fatalf("expected generator result next call, got %q", member.Property)
	}
	requireType[*ast.CallExpression](t, member.Target)
}

func TestParserForOfGeneratorCall(t *testing.T) {
	program := parseProgram(t, `
		function* ids() { yield 1; }
		for (const value of ids()) { print(value); }
	`)
	stmt := requireType[*ast.ForOfStatement](t, program.Statements[1])
	requireType[*ast.CallExpression](t, stmt.Iterable)
}

func TestParserAsyncGeneratorFunctionDeclaration(t *testing.T) {
	program := parseProgram(t, `async function* ids(value) { yield await value; }`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[0])
	if !fn.IsAsync || !fn.IsGenerator {
		t.Fatalf("unexpected async generator declaration: %#v", fn)
	}
}

func TestParserAsyncGeneratorFunctionExpression(t *testing.T) {
	expr := parseExpression(t, `async function* ids(value) { yield await value; }`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if !fn.IsAsync || !fn.IsGenerator || fn.Name != "ids" {
		t.Fatalf("unexpected async generator expression: %#v", fn)
	}
}

func TestParserAnonymousAsyncGeneratorFunctionExpression(t *testing.T) {
	expr := parseExpression(t, `async function*() { yield await next(); }`)
	fn := requireType[*ast.FunctionExpression](t, expr)
	if !fn.IsAsync || !fn.IsGenerator || fn.Name != "" {
		t.Fatalf("unexpected anonymous async generator expression: %#v", fn)
	}
}
