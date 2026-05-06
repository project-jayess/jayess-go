package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersBooleanEqualityComparisons(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{source: `true == true`, want: "i1 1"},
		{source: `true === false`, want: "i1 0"},
		{source: `true != false`, want: "i1 1"},
		{source: `false !== false`, want: "i1 0"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			ir := lowerBooleanComparisonExpressionIR(t, tc.source)
			if !strings.Contains(ir, tc.want) {
				t.Fatalf("expected boolean comparison IR to contain %q:\n%s", tc.want, ir)
			}
			if !strings.Contains(ir, "ret %jayess.value %v0") {
				t.Fatalf("expected boolean comparison IR to return folded value:\n%s", ir)
			}
		})
	}
}

func TestLLVMBackendExpressionEmitterRejectsMixedBooleanComparison(t *testing.T) {
	expr, err := parser.New(lexer.New(`true == 1`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("mixed_boolean_compare", expr); err == nil {
		t.Fatal("expected mixed boolean comparison to be rejected")
	}
}

func TestLLVMBackendExpressionEmitterRejectsBooleanOrderingComparison(t *testing.T) {
	expr, err := parser.New(lexer.New(`true < false`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("boolean_ordering", expr); err == nil {
		t.Fatal("expected boolean ordering comparison to be rejected")
	}
}

func lowerBooleanComparisonExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("boolean_compare", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "boolean-comparison-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
