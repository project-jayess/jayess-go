package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersPositiveNumberLiteral(t *testing.T) {
	ir := lowerUnaryExpressionIR(t, `+42.5`)
	if !strings.Contains(ir, "%v0 = call %jayess.value @jayess_value_from_number(double 42.5)") {
		t.Fatalf("expected positive numeric unary IR:\n%s", ir)
	}
}

func TestLLVMBackendExpressionEmitterLowersNegativeNumberLiteral(t *testing.T) {
	ir := lowerUnaryExpressionIR(t, `-42.5`)
	if !strings.Contains(ir, "%v0 = call %jayess.value @jayess_value_from_number(double -42.5)") {
		t.Fatalf("expected negative numeric unary IR:\n%s", ir)
	}
}

func TestLLVMBackendExpressionEmitterRejectsDynamicNumericUnaryOperand(t *testing.T) {
	program, err := parser.New(lexer.New(`var value = 1; return -value;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements); err == nil {
		t.Fatal("expected dynamic numeric unary operand to be rejected")
	}
}

func lowerUnaryExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("unary", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "numeric-unary-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
