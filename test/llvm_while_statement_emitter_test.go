package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersFalseWhileFallthrough(t *testing.T) {
	ir := lowerWhileStatementIR(t, `while (false) { return 1; } return 4;`)
	for _, want := range []string{
		"br label %while.cond.0",
		"while.cond.0:",
		"%v1 = call %jayess.value @jayess_value_from_boolean(i1 0)",
		"%v2 = call i1 @jayess_value_truthy(%jayess.value %v1)",
		"br i1 %v2, label %while.body.1, label %while.end.2",
		"while.body.1:",
		"%v0 = call %jayess.value @jayess_value_from_number(double 1)",
		"ret %jayess.value %v0",
		"while.end.2:",
		"%v3 = call %jayess.value @jayess_value_from_number(double 4)",
		"ret %jayess.value %v3",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected false while IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersTrueWhileReturn(t *testing.T) {
	ir := lowerWhileStatementIR(t, `while (true) { return 12; }`)
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_number(double 12)",
		"ret %jayess.value %v0",
		"while.cond.0:",
		"br i1 %v2, label %while.body.1, label %while.end.2",
		"while.body.1:",
		"while.end.2:",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected true while IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterRestoresFalseWhileSkippedScope(t *testing.T) {
	ir := lowerWhileStatementIR(t, `var value = 1; while (false) { var value = 2; } return value;`)
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_number(double 1)",
		"%local.0 = alloca %jayess.value",
		"while.body.1:",
		"%v1 = call %jayess.value @jayess_value_from_number(double 2)",
		"%local.1 = alloca %jayess.value",
		"%v4 = load %jayess.value, %jayess.value* %local.0",
		"ret %jayess.value %v4",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected scoped false while IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersDynamicWhileCondition(t *testing.T) {
	ir := lowerWhileStatementIR(t, `var value = true; while (value) { return 1; } return 2;`)
	for _, want := range []string{
		"%v2 = load %jayess.value, %jayess.value* %local.0",
		"%v3 = call i1 @jayess_value_truthy(%jayess.value %v2)",
		"br i1 %v3, label %while.body.1, label %while.end.2",
		"while.body.1:",
		"while.end.2:",
		"%v4 = call %jayess.value @jayess_value_from_number(double 2)",
		"ret %jayess.value %v4",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected dynamic while IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersNonReturningWhileBodyBackEdge(t *testing.T) {
	ir := lowerWhileStatementIR(t, `while (true) { 1; }`)
	for _, want := range []string{
		"while.body.1:",
		"%v0 = call %jayess.value @jayess_value_from_number(double 1)",
		"br label %while.cond.0",
		"while.end.2:",
		"ret %jayess.value undef",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected non-returning while IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerWhileStatementIR(t *testing.T, source string) string {
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
		Name:         "while-statement-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
