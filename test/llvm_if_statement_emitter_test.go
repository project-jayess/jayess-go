package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersTrueIfReturn(t *testing.T) {
	ir := lowerIfStatementIR(t, `if (true) { return 12; }`)
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_boolean(i1 1)",
		"%v1 = call i1 @jayess_value_truthy(%jayess.value %v0)",
		"br i1 %v1, label %if.then.0, label %if.else.1",
		"if.then.0:",
		"%v2 = call %jayess.value @jayess_value_from_number(double 12)",
		"ret %jayess.value %v2",
		"if.else.1:",
		"br label %if.end.2",
		"if.end.2:",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected true if IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersFalseIfAlternativeReturn(t *testing.T) {
	ir := lowerIfStatementIR(t, `if (false) { return 1; } else { return 13; }`)
	for _, want := range []string{
		"%v1 = call i1 @jayess_value_truthy(%jayess.value %v0)",
		"br i1 %v1, label %if.then.0, label %if.else.1",
		"if.then.0:",
		"%v2 = call %jayess.value @jayess_value_from_number(double 1)",
		"ret %jayess.value %v2",
		"if.else.1:",
		"%v3 = call %jayess.value @jayess_value_from_number(double 13)",
		"ret %jayess.value %v3",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected false if alternative IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersFalseIfFallthrough(t *testing.T) {
	ir := lowerIfStatementIR(t, `if (false) { return 1; } return 14;`)
	for _, want := range []string{
		"br i1 %v1, label %if.then.0, label %if.else.1",
		"if.then.0:",
		"%v2 = call %jayess.value @jayess_value_from_number(double 1)",
		"ret %jayess.value %v2",
		"if.else.1:",
		"br label %if.end.2",
		"if.end.2:",
		"%v3 = call %jayess.value @jayess_value_from_number(double 14)",
		"ret %jayess.value %v3",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected false if fallthrough IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterRestoresIfBranchScope(t *testing.T) {
	ir := lowerIfStatementIR(t, `var value = 1; if (true) { var value = 2; } return value;`)
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_number(double 1)",
		"%local.0 = alloca %jayess.value",
		"%v3 = call %jayess.value @jayess_value_from_number(double 2)",
		"%local.1 = alloca %jayess.value",
		"%v4 = load %jayess.value, %jayess.value* %local.0",
		"ret %jayess.value %v4",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected scoped if IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersDynamicIfCondition(t *testing.T) {
	ir := lowerIfStatementIR(t, `var value = true; if (value) { return 1; } return 2;`)
	for _, want := range []string{
		"%v1 = load %jayess.value, %jayess.value* %local.0",
		"%v2 = call i1 @jayess_value_truthy(%jayess.value %v1)",
		"br i1 %v2, label %if.then.0, label %if.else.1",
		"if.end.2:",
		"%v4 = call %jayess.value @jayess_value_from_number(double 2)",
		"ret %jayess.value %v4",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected dynamic if IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerIfStatementIR(t *testing.T, source string) string {
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
		Name:         "if-statement-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
