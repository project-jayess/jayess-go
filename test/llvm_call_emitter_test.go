package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersDirectCallThroughCallableABI(t *testing.T) {
	ir := lowerCallIR(t, `const fn = function (value = 1, ...rest) { return arguments; }; return fn(2, ...[3]);`)
	for _, want := range []string{
		"@jayess_call_function",
		"@jayess_array_new",
		"@jayess_array_push",
		"@jayess_array_spread",
		"call %jayess.value @jayess_call_function",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected direct call IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterLowersMethodCallWithReceiverThis(t *testing.T) {
	ir := lowerCallIR(t, `const object = {run: function () {}}; return object.run(1);`)
	for _, want := range []string{
		"@jayess_value_get_property",
		"call %jayess.value @jayess_value_get_property",
		"call %jayess.value @jayess_call_function",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected method call IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterLowersBindCallApplyHelpers(t *testing.T) {
	for _, source := range []string{
		`const fn = function () {}; return fn.bind(undefined, 1);`,
		`const fn = function () {}; return fn.call(undefined, 1);`,
		`const fn = function () {}; return fn.apply(undefined, [1]);`,
	} {
		ir := lowerCallIR(t, source)
		for _, want := range []string{
			"@jayess_function_new",
			"call %jayess.value @jayess_",
		} {
			if !strings.Contains(ir, want) {
				t.Fatalf("expected helper call IR to contain %q for %s:\n%s", want, source, ir)
			}
		}
	}
}

func lowerCallIR(t *testing.T, source string) string {
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
		Name:         "call-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
