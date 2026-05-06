package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterRestoresShadowedBlockLocal(t *testing.T) {
	ir := lowerBlockScopeIR(t, `var value = 1; { var value = 2; } return value;`)
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_number(double 1)",
		"%local.0 = alloca %jayess.value",
		"%v1 = call %jayess.value @jayess_value_from_number(double 2)",
		"%local.1 = alloca %jayess.value",
		"%v2 = load %jayess.value, %jayess.value* %local.0",
		"ret %jayess.value %v2",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected block scope IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterKeepsOuterAssignmentFromBlock(t *testing.T) {
	ir := lowerBlockScopeIR(t, `var value = 1; { value = 2; } return value;`)
	for _, want := range []string{
		"store %jayess.value %v0, %jayess.value* %local.0",
		"store %jayess.value %v1, %jayess.value* %local.0",
		"%v2 = load %jayess.value, %jayess.value* %local.0",
		"ret %jayess.value %v2",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected block assignment IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterRemovesBlockLocal(t *testing.T) {
	program, err := parser.New(lexer.New(`{ var value = 2; } return value;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements); err == nil {
		t.Fatal("expected block local to be unavailable after block")
	}
}

func lowerBlockScopeIR(t *testing.T, source string) string {
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
		Name:         "block-scope-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
