package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersSwitchDispatch(t *testing.T) {
	ir := lowerSwitchStatementIR(t, `var kind = 2; switch (kind) { case 1: return 1; case 2: return 8; default: return 3; }`)
	for _, want := range []string{
		"declare i1 (%jayess.value, %jayess.value) @jayess_value_strict_equal",
		"br label %switch.check.",
		"switch.check.",
		"call i1 @jayess_value_strict_equal",
		"switch.case.",
		"switch.default.",
		"switch.end.",
		"%v",
		"ret %jayess.value %v",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected switch IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersSwitchFallthrough(t *testing.T) {
	ir := lowerSwitchStatementIR(t, `var value = 1; switch (value) { case 1: value = 4; case 2: return value; default: return 3; }`)
	for _, want := range []string{
		"br label %switch.case.",
		"store %jayess.value",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected switch fallthrough IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerSwitchStatementIR(t *testing.T, source string) string {
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
		Name:         "switch-statement-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
