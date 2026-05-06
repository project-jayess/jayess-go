package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersDoWhileBodyReturn(t *testing.T) {
	ir := lowerDoWhileStatementIR(t, `do { return 9; } while (false);`)
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_number(double 9)",
		"ret %jayess.value %v0",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected do-while return IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersFalseDoWhileFallthrough(t *testing.T) {
	ir := lowerDoWhileStatementIR(t, `var value = 1; do { value = 2; } while (false); return value;`)
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_number(double 1)",
		"%v1 = call %jayess.value @jayess_value_from_number(double 2)",
		"store %jayess.value %v1, %jayess.value* %local.0",
		"do.while.cond.",
		"call i1 @jayess_value_truthy",
		"do.while.body.",
		"do.while.end.",
		"load %jayess.value, %jayess.value* %local.0",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected do-while fallthrough IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterRestoresDoWhileBodyScope(t *testing.T) {
	ir := lowerDoWhileStatementIR(t, `var value = 1; do { var value = 2; } while (false); return value;`)
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_number(double 1)",
		"%local.0 = alloca %jayess.value",
		"%v1 = call %jayess.value @jayess_value_from_number(double 2)",
		"%local.1 = alloca %jayess.value",
		"do.while.cond.",
		"do.while.body.",
		"do.while.end.",
		"load %jayess.value, %jayess.value* %local.0",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected scoped do-while IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersDynamicDoWhileCondition(t *testing.T) {
	ir := lowerDoWhileStatementIR(t, `var keepGoing = false; do { 1; } while (keepGoing);`)
	for _, want := range []string{
		"do.while.cond.",
		"load %jayess.value",
		"call i1 @jayess_value_truthy",
		"br i1",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected dynamic do-while IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersNonReturningConstantDoWhile(t *testing.T) {
	ir := lowerDoWhileStatementIR(t, `do { 1; } while (true);`)
	for _, want := range []string{
		"do.while.cond.",
		"do.while.body.",
		"br label %do.while.cond.",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected non-returning do-while IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerDoWhileStatementIR(t *testing.T, source string) string {
	t.Helper()
	program, err := parser.New(lexer.New(source)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "do-while-statement-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
