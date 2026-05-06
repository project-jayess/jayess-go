package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersForThroughPrimitiveWhile(t *testing.T) {
	ir := lowerForStatementIR(t, `var value = 1; for (; value; value = 0) { value = 2; } return value;`)
	for _, want := range []string{
		"for.cond.",
		"call i1 @jayess_value_truthy",
		"for.body.",
		"store %jayess.value",
		"br label %for.cond.",
		"for.end.",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected for IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersForOfThroughIteratorLoop(t *testing.T) {
	ir := lowerForStatementIR(t, `var last = 0; for (var value of [1, 2]) { last = value; } return last;`)
	for _, want := range []string{
		"call %jayess.value @jayess_for_of_iterator",
		"for.of.cond.",
		"call %jayess.value @jayess_iterator_next",
		"call i1 @jayess_iterator_done",
		"for.of.body.",
		"call %jayess.value @jayess_iterator_value",
		"br label %for.of.cond.",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected for-of IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersForInThroughIteratorLoop(t *testing.T) {
	ir := lowerForStatementIR(t, `var last = ""; for (var key in { name: 1 }) { last = key; } return last;`)
	for _, want := range []string{
		"call %jayess.value @jayess_for_in_iterator",
		"for.in.cond.",
		"call %jayess.value @jayess_iterator_next",
		"call i1 @jayess_iterator_done",
		"for.in.body.",
		"call %jayess.value @jayess_iterator_value",
		"br label %for.in.cond.",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected for-in IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerForStatementIR(t *testing.T, source string) string {
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
		Name:         "for-statement-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
