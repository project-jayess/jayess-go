package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersThrowThroughSharedTarget(t *testing.T) {
	ir := lowerThrowAbruptIR(t, `if (true) { throw 1; } throw 2;`)
	for _, want := range []string{
		"%local.0 = alloca %jayess.value",
		"store %jayess.value",
		"br label %throw.",
		"throw.",
		"load %jayess.value, %jayess.value* %local.0",
		"call void @jayess_throw_unhandled",
		"ret %jayess.value undef",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected shared throw IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerThrowAbruptIR(t *testing.T, source string) string {
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
		Name:         "throw-abrupt-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
