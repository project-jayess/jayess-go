package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersBigIntNegateLiteral(t *testing.T) {
	ir := lowerBigIntUnaryExpressionIR(t, `-12n`)
	for _, want := range []string{
		`@.jayess.literal.0 = private unnamed_addr constant [4 x i8] c"-12\00"`,
		"%v0 = call %jayess.value @jayess_value_from_bigint_string",
		"ret %jayess.value %v0",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected BigInt negate IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterLowersBigIntBitNotLiteral(t *testing.T) {
	ir := lowerBigIntUnaryExpressionIR(t, `~12n`)
	for _, want := range []string{
		`@.jayess.literal.0 = private unnamed_addr constant [4 x i8] c"-13\00"`,
		"%v0 = call %jayess.value @jayess_value_from_bigint_string",
		"ret %jayess.value %v0",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected BigInt bitwise-not IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterRejectsPositiveBigIntLiteral(t *testing.T) {
	expr, err := parser.New(lexer.New(`+12n`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("positive_bigint", expr); err == nil {
		t.Fatal("expected positive BigInt unary expression to be rejected")
	}
}

func TestLLVMBackendExpressionEmitterRejectsDynamicBigIntBitNotOperand(t *testing.T) {
	program, err := parser.New(lexer.New(`var value = 12n; return ~value;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements); err == nil {
		t.Fatal("expected dynamic BigInt bitwise-not operand to be rejected")
	}
}

func lowerBigIntUnaryExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("bigint_unary", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "bigint-unary-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
