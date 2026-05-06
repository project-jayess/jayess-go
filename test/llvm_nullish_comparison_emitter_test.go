package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersNullishEqualityComparisons(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{source: `null == undefined`, want: "i1 1"},
		{source: `null === undefined`, want: "i1 0"},
		{source: `null != undefined`, want: "i1 0"},
		{source: `undefined !== null`, want: "i1 1"},
		{source: `null === null`, want: "i1 1"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			ir := lowerNullishComparisonExpressionIR(t, tc.source)
			if !strings.Contains(ir, tc.want) {
				t.Fatalf("expected nullish comparison IR to contain %q:\n%s", tc.want, ir)
			}
			if !strings.Contains(ir, "ret %jayess.value %v0") {
				t.Fatalf("expected nullish comparison IR to return folded value:\n%s", ir)
			}
		})
	}
}

func TestLLVMBackendExpressionEmitterRejectsMixedNullishComparison(t *testing.T) {
	expr, err := parser.New(lexer.New(`null == 0`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("mixed_nullish_compare", expr); err == nil {
		t.Fatal("expected mixed nullish comparison to be rejected")
	}
}

func TestLLVMBackendExpressionEmitterRejectsNullishOrderingComparison(t *testing.T) {
	expr, err := parser.New(lexer.New(`null < undefined`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("nullish_ordering", expr); err == nil {
		t.Fatal("expected nullish ordering comparison to be rejected")
	}
}

func lowerNullishComparisonExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("nullish_compare", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "nullish-comparison-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
