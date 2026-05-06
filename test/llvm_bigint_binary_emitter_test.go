package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersBigIntBinaryLiterals(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{source: "12n + 13n", want: `c"25\00"`},
		{source: "12n - 13n", want: `c"-1\00"`},
		{source: "12n * 13n", want: `c"156\00"`},
		{source: "13n / 5n", want: `c"2\00"`},
		{source: "13n % 5n", want: `c"3\00"`},
		{source: "2n ** 8n", want: `c"256\00"`},
		{source: "12n & 10n", want: `c"8\00"`},
		{source: "12n | 1n", want: `c"13\00"`},
		{source: "5n ^ 3n", want: `c"6\00"`},
		{source: "5n << 2n", want: `c"20\00"`},
		{source: "64n >> 3n", want: `c"8\00"`},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			ir := lowerBigIntBinaryExpressionIR(t, tc.source)
			if !strings.Contains(ir, tc.want) {
				t.Fatalf("expected BigInt binary IR to contain %q:\n%s", tc.want, ir)
			}
			if !strings.Contains(ir, "@jayess_value_from_bigint_string") {
				t.Fatalf("expected BigInt binary IR to construct BigInt value:\n%s", ir)
			}
			if !strings.Contains(ir, "ret %jayess.value %v0") {
				t.Fatalf("expected BigInt binary IR to return folded value:\n%s", ir)
			}
		})
	}
}

func TestLLVMBackendExpressionEmitterRejectsMixedBigIntBinaryOperand(t *testing.T) {
	expr, err := parser.New(lexer.New(`12n + 12`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("mixed_bigint_binary", expr); err == nil {
		t.Fatal("expected mixed BigInt binary operand to be rejected")
	}
}

func TestLLVMBackendExpressionEmitterRejectsBigIntDivisionByZero(t *testing.T) {
	expr, err := parser.New(lexer.New(`12n / 0n`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("bigint_division_zero", expr); err == nil {
		t.Fatal("expected BigInt division by zero to be rejected")
	}
}

func TestLLVMBackendExpressionEmitterRejectsDynamicBigIntBinaryOperand(t *testing.T) {
	program, err := parser.New(lexer.New(`var value = 12n; return value + 13n;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements); err == nil {
		t.Fatal("expected dynamic BigInt binary operand to be rejected")
	}
}

func lowerBigIntBinaryExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("bigint_binary_expr", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "bigint-binary-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
