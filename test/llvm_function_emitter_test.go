package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersFunctionExpressionToCallableValue(t *testing.T) {
	ir := lowerFunctionIR(t, `const fn = function () {}; return fn;`)
	for _, want := range []string{
		"@jayess_function_new",
		"call %jayess.value @jayess_function_new()",
		"store %jayess.value %v0, %jayess.value* %local.0",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected function expression IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersFunctionDeclarationToCallableBinding(t *testing.T) {
	ir := lowerFunctionIR(t, `function helper() { return 1; } return helper;`)
	for _, want := range []string{
		"@jayess_function_new",
		"call %jayess.value @jayess_function_new()",
		"%local.0 = alloca %jayess.value",
		"store %jayess.value %v0, %jayess.value* %local.0",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected function declaration IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterLowersArrowFunctionToCallableValue(t *testing.T) {
	ir := lowerFunctionIR(t, `const fn = () => 1; return fn;`)
	if !strings.Contains(ir, "call %jayess.value @jayess_function_new()") {
		t.Fatalf("expected arrow function IR to create callable value:\n%s", ir)
	}
}

func lowerFunctionIR(t *testing.T, source string) string {
	t.Helper()
	program, err := parser.New(lexer.New(source)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "function-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
