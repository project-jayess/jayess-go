package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersCommaExpressionLeftToRight(t *testing.T) {
	expr, err := parser.New(lexer.New(`1, 2`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("comma", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "comma-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_number(double 1)",
		"%v1 = call %jayess.value @jayess_value_from_number(double 2)",
		"ret %jayess.value %v1",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected comma IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterReturnsCommaExpressionRightValue(t *testing.T) {
	program, err := parser.New(lexer.New(`return false, true;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "comma-statement-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_boolean(i1 0)",
		"%v1 = call %jayess.value @jayess_value_from_boolean(i1 1)",
		"ret %jayess.value %v1",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected statement comma IR to contain %q:\n%s", want, ir)
		}
	}
}
