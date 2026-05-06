package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersDeleteThroughRuntimeCalls(t *testing.T) {
	cases := []struct {
		name string
		code string
		want string
	}{
		{name: "value", code: `delete value++`, want: "@jayess_value_delete_value"},
		{name: "member", code: `delete value.name`, want: "@jayess_value_delete_property"},
		{name: "index", code: `delete value[0]`, want: "@jayess_value_delete_index"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ir := lowerRuntimeOperatorProgramIR(t, `var value = 1; return `+tc.code+`;`)
			if !strings.Contains(ir, tc.want) {
				t.Fatalf("expected delete IR to contain %q:\n%s", tc.want, ir)
			}
		})
	}
}

func TestLLVMBackendExpressionEmitterLowersInThroughRuntimeCall(t *testing.T) {
	ir := lowerRuntimeOperatorProgramIR(t, `var object = 1; return "name" in object;`)
	for _, want := range []string{
		"declare %jayess.value (%jayess.value, %jayess.value) @jayess_value_in",
		"call %jayess.value @jayess_value_in",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected in-operator IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterLowersInstanceofThroughRuntimeCall(t *testing.T) {
	ir := lowerRuntimeOperatorProgramIR(t, `var value = 1; var Type = 2; return value instanceof Type;`)
	for _, want := range []string{
		"declare %jayess.value (%jayess.value, %jayess.value) @jayess_value_instanceof",
		"call %jayess.value @jayess_value_instanceof",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected instanceof IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerRuntimeOperatorProgramIR(t *testing.T, source string) string {
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
		Name:         "runtime-operator-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
