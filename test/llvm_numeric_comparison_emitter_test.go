package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersNumericComparisonLiterals(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{source: "1 == 1", want: "i1 1"},
		{source: "1 === 2", want: "i1 0"},
		{source: "1 != 2", want: "i1 1"},
		{source: "1 !== 1", want: "i1 0"},
		{source: "1 < 2", want: "i1 1"},
		{source: "1 <= 1", want: "i1 1"},
		{source: "3 > 2", want: "i1 1"},
		{source: "3 >= 4", want: "i1 0"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			ir := lowerNumericComparisonExpressionIR(t, tc.source)
			if !strings.Contains(ir, tc.want) {
				t.Fatalf("expected numeric comparison IR to contain %q:\n%s", tc.want, ir)
			}
			if !strings.Contains(ir, "ret %jayess.value %v0") {
				t.Fatalf("expected numeric comparison IR to return folded value:\n%s", ir)
			}
		})
	}
}

func TestLLVMBackendExpressionEmitterRejectsDynamicNumericComparisonOperand(t *testing.T) {
	program, err := parser.New(lexer.New(`var value = 1; return value < 2;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements); err == nil {
		t.Fatal("expected dynamic numeric comparison operand to be rejected")
	}
}

func TestLLVMBackendExpressionEmitterRejectsUnsupportedComparisonOperator(t *testing.T) {
	expr, err := parser.New(lexer.New(`1 in items`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("bad_comparison", expr); err == nil {
		t.Fatal("expected unsupported comparison operator to be rejected")
	}
}

func lowerNumericComparisonExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("comparison_expr", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "numeric-comparison-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
