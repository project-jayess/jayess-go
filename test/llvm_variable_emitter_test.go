package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersPrimitiveVariableReturn(t *testing.T) {
	program, err := parser.New(lexer.New(`const answer = 42; return answer;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	module := llvmbackend.Module{
		Name:         "variable-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_number(double 42)",
		"%local.0 = alloca %jayess.value",
		"store %jayess.value %v0, %jayess.value* %local.0",
		"%v1 = load %jayess.value, %jayess.value* %local.0",
		"ret %jayess.value %v1",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected variable IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersUninitializedVariableAsUndefinedValue(t *testing.T) {
	program, err := parser.New(lexer.New(`var missing; return missing;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	module := llvmbackend.Module{
		Name:         "variable-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"%local.0 = alloca %jayess.value",
		"store %jayess.value undef, %jayess.value* %local.0",
		"%v0 = load %jayess.value, %jayess.value* %local.0",
		"ret %jayess.value %v0",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected uninitialized variable IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterRejectsUnknownIdentifier(t *testing.T) {
	program, err := parser.New(lexer.New(`return missing;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements); err == nil {
		t.Fatal("expected unknown emitted local to be rejected")
	}
}
