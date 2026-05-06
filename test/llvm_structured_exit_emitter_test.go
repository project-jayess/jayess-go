package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersLoopBreakToExitStack(t *testing.T) {
	ir := lowerStructuredExitIR(t, `while (true) { break; } return 1;`)
	for _, want := range []string{
		"while.body.",
		"br label %while.end.",
		"while.end.",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected loop break IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersLoopContinueToExitStack(t *testing.T) {
	ir := lowerStructuredExitIR(t, `var value = 1; for (; value; value = 0) { continue; } return value;`)
	for _, want := range []string{
		"for.body.",
		"br label %for.update.",
		"for.update.",
		"store %jayess.value",
		"br label %for.cond.",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected loop continue IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersSwitchBreakToExitStack(t *testing.T) {
	ir := lowerStructuredExitIR(t, `var value = 1; switch (value) { case 1: value = 2; break; default: value = 3; } return value;`)
	for _, want := range []string{
		"switch.case.",
		"br label %switch.end.",
		"switch.end.",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected switch break IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerStructuredExitIR(t *testing.T, source string) string {
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
		Name:         "structured-exit-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
