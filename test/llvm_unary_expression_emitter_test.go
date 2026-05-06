package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersVoidExpression(t *testing.T) {
	expr, err := parser.New(lexer.New(`void 1`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("void_expr", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "void-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_number(double 1)",
		"%v1 = call %jayess.value @jayess_value_undefined()",
		"ret %jayess.value %v1",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected void IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterReturnsVoidExpression(t *testing.T) {
	program, err := parser.New(lexer.New(`return void false;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "void-statement-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_boolean(i1 0)",
		"%v1 = call %jayess.value @jayess_value_undefined()",
		"ret %jayess.value %v1",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected statement void IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterRejectsUnsupportedUnaryExpression(t *testing.T) {
	program, err := parser.New(lexer.New(`var value = true; return !value;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeStatementFunction("bad_unary", program.Statements); err == nil {
		t.Fatal("expected unsupported unary expression to be rejected")
	}
}
