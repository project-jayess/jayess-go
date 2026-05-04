package test

import (
	"testing"

	"jayess-go/ast"
	"jayess-go/lexer"
	"jayess-go/parser"
)

func TestParserProgramVariableDeclarations(t *testing.T) {
	program := parseProgram(t, "var total = 1 + 2 * 3;\nconst label = \"ready\";")
	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}

	first := requireType[*ast.VariableDecl](t, program.Statements[0])
	if first.Kind != ast.DeclarationVar || first.Name != "total" {
		t.Fatalf("unexpected first declaration: %#v", first)
	}
	requireType[*ast.BinaryExpression](t, first.Value)

	second := requireType[*ast.VariableDecl](t, program.Statements[1])
	if second.Kind != ast.DeclarationConst || second.Name != "label" {
		t.Fatalf("unexpected second declaration: %#v", second)
	}
	requireType[*ast.StringLiteral](t, second.Value)
}

func TestParserProgramAllowsNewlineTerminators(t *testing.T) {
	program := parseProgram(t, "var first = 1\nvar second = 2")
	if len(program.Statements) != 2 {
		t.Fatalf("expected newline to terminate first declaration, got %d statements", len(program.Statements))
	}
}

func TestParserProgramAllowsEmptyStatements(t *testing.T) {
	program := parseProgram(t, `; var value = 1; ;`)
	if len(program.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(program.Statements))
	}
	requireType[*ast.EmptyStatement](t, program.Statements[0])
	requireType[*ast.VariableDecl](t, program.Statements[1])
	requireType[*ast.EmptyStatement](t, program.Statements[2])
}

func TestParserBlockAllowsEmptyStatements(t *testing.T) {
	program := parseProgram(t, `{ ; var value = 1; }`)
	block := requireType[*ast.BlockStatement](t, program.Statements[0])
	if len(block.Statements) != 2 {
		t.Fatalf("expected 2 block statements, got %d", len(block.Statements))
	}
	requireType[*ast.EmptyStatement](t, block.Statements[0])
	requireType[*ast.VariableDecl](t, block.Statements[1])
}

func TestParserProgramRejectsMissingTerminatorOnSameLine(t *testing.T) {
	_, err := parser.New(lexer.New("var first = 1 var second = 2")).ParseProgram()
	if err == nil {
		t.Fatalf("expected missing terminator error")
	}
}

func TestParserProgramRejectsConstWithoutInitializer(t *testing.T) {
	_, err := parser.New(lexer.New("const value;")).ParseProgram()
	if err == nil {
		t.Fatalf("expected const initializer error")
	}
}

func parseProgramError(source string) (*ast.Program, error) {
	return parser.New(lexer.New(source)).ParseProgram()
}

func parseProgram(t *testing.T, source string) *ast.Program {
	t.Helper()
	program, err := parser.New(lexer.New(source)).ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram(%q) returned error: %v", source, err)
	}
	return program
}
