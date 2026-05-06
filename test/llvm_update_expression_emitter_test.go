package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersLocalUpdateExpression(t *testing.T) {
	ir := lowerUpdateProgramIR(t, `var value = 1; value++; return value;`)
	for _, want := range []string{
		"%v1 = load %jayess.value, %jayess.value* %local.0",
		"%v2 = call %jayess.value @jayess_value_update_increment(%jayess.value %v1)",
		"store %jayess.value %v2, %jayess.value* %local.0",
		"%v3 = load %jayess.value, %jayess.value* %local.0",
		"ret %jayess.value %v3",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected local update IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterLowersPrefixUpdateResult(t *testing.T) {
	ir := lowerUpdateProgramIR(t, `var value = 1; var next = ++value; return next;`)
	for _, want := range []string{
		"%v1 = load %jayess.value, %jayess.value* %local.0",
		"%v2 = call %jayess.value @jayess_value_update_increment(%jayess.value %v1)",
		"store %jayess.value %v2, %jayess.value* %local.0",
		"store %jayess.value %v2, %jayess.value* %local.1",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected prefix update IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterLowersMemberAndIndexUpdates(t *testing.T) {
	cases := []struct {
		name string
		code string
		want []string
	}{
		{
			name: "member",
			code: `var value = 1; value.count++;`,
			want: []string{
				"call %jayess.value @jayess_value_get_property",
				"call %jayess.value @jayess_value_update_increment",
				"call void @jayess_value_set_property",
			},
		},
		{
			name: "index",
			code: `var values = 1; values[0]--;`,
			want: []string{
				"call %jayess.value @jayess_value_get_index",
				"call %jayess.value @jayess_value_update_decrement",
				"call void @jayess_value_set_index",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ir := lowerUpdateProgramIR(t, tc.code)
			for _, want := range tc.want {
				if !strings.Contains(ir, want) {
					t.Fatalf("expected update IR to contain %q:\n%s", want, ir)
				}
			}
		})
	}
}

func lowerUpdateProgramIR(t *testing.T, source string) string {
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
		Name:         "update-expression-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
