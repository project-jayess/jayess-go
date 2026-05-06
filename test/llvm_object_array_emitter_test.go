package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersObjectConstruction(t *testing.T) {
	ir := lowerObjectArrayExpressionIR(t, `({ name: "jayess", ["version"]: 1, ...extra })`)
	for _, want := range []string{
		"call %jayess.value @jayess_object_new()",
		"call void @jayess_value_set_property",
		"call void @jayess_object_spread",
		"ret %jayess.value %v1",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected object construction IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterLowersArrayConstruction(t *testing.T) {
	ir := lowerObjectArrayExpressionIR(t, `[1, , ...extra, 4]`)
	for _, want := range []string{
		"call %jayess.value @jayess_array_new()",
		"call void @jayess_array_push",
		"call void @jayess_array_elide",
		"call void @jayess_array_spread",
		"ret %jayess.value %v1",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected array construction IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerObjectArrayExpressionIR(t *testing.T, source string) string {
	t.Helper()
	program, err := parser.New(lexer.New(`var extra = 1; return ` + source + `;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "object-array-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
