package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersNumericBinaryLiterals(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{source: "1 + 2", want: "double 3"},
		{source: "7 - 2", want: "double 5"},
		{source: "3 * 4", want: "double 12"},
		{source: "8 / 2", want: "double 4"},
		{source: "8 % 3", want: "double 2"},
		{source: "2 ** 3", want: "double 8"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			ir := lowerNumericBinaryExpressionIR(t, tc.source)
			if !strings.Contains(ir, tc.want) {
				t.Fatalf("expected numeric binary IR to contain %q:\n%s", tc.want, ir)
			}
			if !strings.Contains(ir, "ret %jayess.value %v0") {
				t.Fatalf("expected numeric binary IR to return folded value:\n%s", ir)
			}
		})
	}
}

func TestLLVMBackendExpressionEmitterRejectsDynamicNumericBinaryOperand(t *testing.T) {
	program, err := parser.New(lexer.New(`var value = 1; return value + 2;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements); err == nil {
		t.Fatal("expected dynamic numeric binary operand to be rejected")
	}
}

func TestLLVMBackendExpressionEmitterRejectsUnsupportedBinaryOperator(t *testing.T) {
	expr, err := parser.New(lexer.New(`1 & 2`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("bad_binary", expr); err == nil {
		t.Fatal("expected unsupported binary operator to be rejected")
	}
}

func lowerNumericBinaryExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("binary_expr", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "numeric-binary-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
