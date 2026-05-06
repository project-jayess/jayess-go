package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersPrimitiveReturn(t *testing.T) {
	program, err := parser.New(lexer.New(`return "jayess";`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	module := llvmbackend.Module{
		Name:         "statement-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		`@.jayess.literal.0 = private unnamed_addr constant [7 x i8] c"jayess\00"`,
		"%v0 = call %jayess.value @jayess_value_from_string_copy",
		"ret %jayess.value %v0",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected statement IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersPrimitiveExpressionStatement(t *testing.T) {
	program, err := parser.New(lexer.New(`true;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	module := llvmbackend.Module{
		Name:         "statement-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_boolean(i1 1)",
		"ret %jayess.value undef",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected expression statement IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterRejectsUnsupportedStatement(t *testing.T) {
	program, err := parser.New(lexer.New(`debugger;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements); err == nil {
		t.Fatal("expected unsupported debugger statement to be rejected")
	}
}

func TestLLVMBackendStatementEmitterRejectsUnreachableStatement(t *testing.T) {
	program, err := parser.New(lexer.New(`return null; true;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements); err == nil {
		t.Fatal("expected statement after return to be rejected")
	}
}
