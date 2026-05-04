package test

import (
	"testing"

	"jayess-go/ast"
	"jayess-go/lexer"
	"jayess-go/parser"
)

func TestParserForAwaitOfStatement(t *testing.T) {
	program := parseProgram(t, `for await (const item of items) { print(item); }`)
	stmt := requireType[*ast.ForOfStatement](t, program.Statements[0])
	if !stmt.Await || stmt.Kind != ast.DeclarationConst || stmt.Name != "item" {
		t.Fatalf("unexpected for await...of statement: %#v", stmt)
	}
	requireType[*ast.Identifier](t, stmt.Iterable)
}

func TestParserForAwaitOfAsyncGeneratorCall(t *testing.T) {
	program := parseProgram(t, `
		async function* ids() { yield 1; }
		async function run() {
			for await (const value of ids()) { print(value); }
		}
	`)
	fn := requireType[*ast.FunctionDecl](t, program.Statements[1])
	stmt := requireType[*ast.ForOfStatement](t, fn.Body[0])
	if !stmt.Await {
		t.Fatalf("expected for await...of statement")
	}
	requireType[*ast.CallExpression](t, stmt.Iterable)
}

func TestParserRejectsForAwaitInStatement(t *testing.T) {
	_, err := parser.New(lexer.New(`for await (const key in item) { print(key); }`)).ParseProgram()
	if err == nil {
		t.Fatalf("expected for await...in parse error")
	}
}
