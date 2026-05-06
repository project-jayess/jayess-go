package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersReturnThroughSharedTarget(t *testing.T) {
	ir := lowerReturnAbruptIR(t, `if (true) { return 1; } return 2;`)
	for _, want := range []string{
		"%local.0 = alloca %jayess.value",
		"store %jayess.value",
		"br label %return.",
		"return.",
		"load %jayess.value, %jayess.value* %local.0",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected shared return IR to contain %q:\n%s", want, ir)
		}
	}
	if strings.Count(ir, "\n  ret %jayess.value ") != 1 {
		t.Fatalf("expected exactly one concrete runtime return instruction:\n%s", ir)
	}
}

func lowerReturnAbruptIR(t *testing.T, source string) string {
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
		Name:         "return-abrupt-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
