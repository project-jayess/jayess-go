package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersBigIntComparisonLiterals(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{source: "12n == 12n", want: "i1 1"},
		{source: "12n === 13n", want: "i1 0"},
		{source: "12n != 13n", want: "i1 1"},
		{source: "12n !== 12n", want: "i1 0"},
		{source: "12n < 13n", want: "i1 1"},
		{source: "13n <= 13n", want: "i1 1"},
		{source: "14n > 13n", want: "i1 1"},
		{source: "14n >= 15n", want: "i1 0"},
		{source: "100000000000000000000n > 99999999999999999999n", want: "i1 1"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			ir := lowerBigIntComparisonExpressionIR(t, tc.source)
			if !strings.Contains(ir, tc.want) {
				t.Fatalf("expected BigInt comparison IR to contain %q:\n%s", tc.want, ir)
			}
			if !strings.Contains(ir, "ret %jayess.value %v0") {
				t.Fatalf("expected BigInt comparison IR to return folded value:\n%s", ir)
			}
		})
	}
}

func TestLLVMBackendExpressionEmitterRejectsMixedBigIntComparisonOperand(t *testing.T) {
	expr, err := parser.New(lexer.New(`12n == 12`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("mixed_bigint_comparison", expr); err == nil {
		t.Fatal("expected mixed BigInt comparison operand to be rejected")
	}
}

func TestLLVMBackendExpressionEmitterRejectsDynamicBigIntComparisonOperand(t *testing.T) {
	program, err := parser.New(lexer.New(`var value = 12n; return value < 13n;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements); err == nil {
		t.Fatal("expected dynamic BigInt comparison operand to be rejected")
	}
}

func lowerBigIntComparisonExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("bigint_comparison_expr", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "bigint-comparison-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
