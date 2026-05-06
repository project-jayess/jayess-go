package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersLabeledLoopBreak(t *testing.T) {
	ir := lowerLabeledExitIR(t, `outer: while (true) { break outer; } return 1;`)
	for _, want := range []string{
		"while.body.",
		"br label %while.end.",
		"while.end.",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected labeled loop break IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersLabeledLoopContinue(t *testing.T) {
	ir := lowerLabeledExitIR(t, `var value = 1; outer: for (; value; value = 0) { continue outer; } return value;`)
	for _, want := range []string{
		"for.body.",
		"br label %for.update.",
		"for.update.",
		"br label %for.cond.",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected labeled loop continue IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersLabeledBlockBreak(t *testing.T) {
	ir := lowerLabeledExitIR(t, `done: { break done; } return 2;`)
	for _, want := range []string{
		"br label %label.done.end.",
		"label.done.end.",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected labeled block break IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerLabeledExitIR(t *testing.T, source string) string {
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
		Name:         "labeled-exit-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
