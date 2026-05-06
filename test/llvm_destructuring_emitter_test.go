package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersArrayDestructuringDeclaration(t *testing.T) {
	ir := lowerDestructuringIR(t, `var [first, second = 2] = [1]; return first;`)
	for _, want := range []string{
		"@jayess_destructure_array_index",
		"@jayess_destructure_default",
		"call %jayess.value @jayess_destructure_array_index",
		"call %jayess.value @jayess_destructure_default",
		"store %jayess.value %v",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected array destructuring IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersObjectDestructuringDeclaration(t *testing.T) {
	ir := lowerDestructuringIR(t, `var {name} = {name: 3}; return name;`)
	for _, want := range []string{
		"@jayess_destructure_property",
		"call %jayess.value @jayess_destructure_property",
		"store %jayess.value %v",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected object destructuring IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersDestructuredCatchBinding(t *testing.T) {
	ir := lowerDestructuringIR(t, `try { throw {code: 4}; } catch ({code}) { return code; }`)
	for _, want := range []string{
		"try.catch.",
		"load %jayess.value, %jayess.value* %local.0",
		"@jayess_destructure_property",
		"call %jayess.value @jayess_destructure_property",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected destructured catch IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersForOfDestructuringBinding(t *testing.T) {
	ir := lowerDestructuringIR(t, `for (var [value] of [[5]]) { return value; }`)
	for _, want := range []string{
		"@jayess_for_of_iterator",
		"@jayess_destructure_array_index",
		"call %jayess.value @jayess_destructure_array_index",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected for-of destructuring IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerDestructuringIR(t *testing.T, source string) string {
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
		Name:         "destructuring-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
