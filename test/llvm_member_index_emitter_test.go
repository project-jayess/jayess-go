package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersMemberAccess(t *testing.T) {
	ir := lowerMemberIndexExpressionIR(t, `value.name`)
	for _, want := range []string{
		"@jayess_value_get_property",
		"call %jayess.value @jayess_value_get_property",
		"ret %jayess.value %v",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected member access IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterLowersIndexAccess(t *testing.T) {
	ir := lowerMemberIndexExpressionIR(t, `value[0]`)
	for _, want := range []string{
		"@jayess_value_get_index",
		"call %jayess.value @jayess_value_get_index",
		"ret %jayess.value %v",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected index access IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerMemberIndexExpressionIR(t *testing.T, source string) string {
	t.Helper()
	program, err := parser.New(lexer.New(`var value = 1; return ` + source + `;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "member-index-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
