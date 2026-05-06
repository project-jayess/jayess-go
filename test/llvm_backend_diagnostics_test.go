package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionErrorIncludesSourcePosition(t *testing.T) {
	expr, err := parser.New(lexer.New("\n  name")).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	_, _, _, err = llvmbackend.LowerRuntimeExpressionFunction("bad", expr)
	if err == nil {
		t.Fatal("expected unsupported expression to be rejected")
	}
	if !strings.HasPrefix(err.Error(), "2:3: ") {
		t.Fatalf("expected expression diagnostic to include source position, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "undefined emitted local name") {
		t.Fatalf("expected expression diagnostic to preserve backend error, got %q", err.Error())
	}
}

func TestLLVMBackendStatementErrorIncludesSourcePosition(t *testing.T) {
	program, err := parser.New(lexer.New("\ndebugger;")).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	_, _, _, err = llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err == nil {
		t.Fatal("expected unsupported statement to be rejected")
	}
	if !strings.HasPrefix(err.Error(), "2:1: ") {
		t.Fatalf("expected statement diagnostic to include source position, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "unsupported runtime statement") {
		t.Fatalf("expected statement diagnostic to preserve backend error, got %q", err.Error())
	}
}

func TestLLVMBackendNestedExpressionErrorKeepsExpressionPosition(t *testing.T) {
	program, err := parser.New(lexer.New("\nreturn unknown;")).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	_, _, _, err = llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err == nil {
		t.Fatal("expected unsupported expression to be rejected")
	}
	if !strings.HasPrefix(err.Error(), "2:8: ") {
		t.Fatalf("expected nested expression diagnostic to keep expression position, got %q", err.Error())
	}
	if strings.Count(err.Error(), ": undefined emitted local unknown") != 1 {
		t.Fatalf("expected diagnostic to wrap backend error once, got %q", err.Error())
	}
}
