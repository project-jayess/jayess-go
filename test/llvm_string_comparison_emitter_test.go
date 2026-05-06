package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersStringEqualityComparisons(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{source: `"a" == "a"`, want: "i1 1"},
		{source: `"a" === "b"`, want: "i1 0"},
		{source: `"a" != "b"`, want: "i1 1"},
		{source: `"a" !== "a"`, want: "i1 0"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			ir := lowerStringComparisonExpressionIR(t, tc.source)
			if !strings.Contains(ir, tc.want) {
				t.Fatalf("expected string comparison IR to contain %q:\n%s", tc.want, ir)
			}
			if !strings.Contains(ir, "ret %jayess.value %v0") {
				t.Fatalf("expected string comparison IR to return folded value:\n%s", ir)
			}
		})
	}
}

func TestLLVMBackendExpressionEmitterRejectsMixedStringComparison(t *testing.T) {
	expr, err := parser.New(lexer.New(`"a" == 1`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("mixed_string_compare", expr); err == nil {
		t.Fatal("expected mixed string comparison to be rejected")
	}
}

func TestLLVMBackendExpressionEmitterRejectsStringOrderingComparison(t *testing.T) {
	expr, err := parser.New(lexer.New(`"a" < "b"`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("string_ordering", expr); err == nil {
		t.Fatal("expected string ordering comparison to be rejected")
	}
}

func lowerStringComparisonExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("string_compare", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "string-comparison-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
