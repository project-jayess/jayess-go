package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersClosureCapturesThroughEnvironment(t *testing.T) {
	ir := lowerClosureProgramIR(t, `const value = 7; const fn = () => value; return fn;`)
	for _, want := range []string{
		"@jayess_closure_environment_new",
		"@jayess_closure_environment_set",
		"@jayess_function_new_with_closure",
		"call i8* @jayess_closure_environment_new()",
		"call void @jayess_closure_environment_set",
		"call %jayess.value @jayess_function_new_with_closure",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected closure IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterDoesNotAllocateClosureForUncapturingFunction(t *testing.T) {
	ir := lowerClosureProgramIR(t, `const fn = () => 1; return fn;`)
	if strings.Contains(ir, "jayess_closure_environment_new") {
		t.Fatalf("did not expect closure environment for uncapturing function:\n%s", ir)
	}
	if !strings.Contains(ir, "call %jayess.value @jayess_function_new()") {
		t.Fatalf("expected normal function allocation:\n%s", ir)
	}
}

func lowerClosureProgramIR(t *testing.T, source string) string {
	t.Helper()
	program, err := parser.New(lexer.New(source)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeProgramFunction("main", program)
	if err != nil {
		t.Fatalf("lower program: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "closure-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
