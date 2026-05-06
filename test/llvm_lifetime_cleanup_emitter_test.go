package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendEmitsReleaseForLifetimePlannedBlockCleanup(t *testing.T) {
	ir := lowerLifetimeProgramIR(t, `{ const value = {}; } return 1;`)
	for _, want := range []string{
		"@jayess_value_release",
		"call void @jayess_value_release",
		"%v1 = load %jayess.value, %jayess.value* %local.0",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected lifetime cleanup IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendEmitsCleanupBeforeAbruptReturn(t *testing.T) {
	ir := lowerLifetimeProgramIR(t, `{ const value = {}; return 1; }`)
	releaseIndex := strings.Index(ir, "call void @jayess_value_release")
	branchIndex := strings.Index(ir, "br label %return.")
	if releaseIndex < 0 || branchIndex < 0 || releaseIndex > branchIndex {
		t.Fatalf("expected cleanup release before return branch:\n%s", ir)
	}
}

func TestLLVMBackendEmitsRetainForExtendedLifetime(t *testing.T) {
	ir := lowerLifetimeProgramIR(t, `const value = {}; return value;`)
	if !strings.Contains(ir, "@jayess_value_retain") {
		t.Fatalf("expected retained extended lifetime declaration:\n%s", ir)
	}
}

func lowerLifetimeProgramIR(t *testing.T, source string) string {
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
		Name:         "lifetime-cleanup-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
