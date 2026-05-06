package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersStringLiteralConcatenation(t *testing.T) {
	ir := lowerStringConcatExpressionIR(t, `"jay" + "ess"`)
	for _, want := range []string{
		`c"jayess\00"`,
		"%v0 = call %jayess.value @jayess_value_from_string_copy",
		"ret %jayess.value %v0",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected string concat IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterRejectsMixedStringConcatenation(t *testing.T) {
	expr, err := parser.New(lexer.New(`"count" + 1`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("mixed_concat", expr); err == nil {
		t.Fatal("expected mixed string concatenation to be rejected")
	}
}

func TestLLVMBackendExpressionEmitterKeepsNumericAddition(t *testing.T) {
	ir := lowerStringConcatExpressionIR(t, `1 + 2`)
	if !strings.Contains(ir, "%v0 = call %jayess.value @jayess_value_from_number(double 3)") {
		t.Fatalf("expected numeric addition to remain folded:\n%s", ir)
	}
}

func lowerStringConcatExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("concat_expr", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "string-concat-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
