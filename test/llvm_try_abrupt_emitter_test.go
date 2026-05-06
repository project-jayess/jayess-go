package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersTryCatchThroughThrowHandler(t *testing.T) {
	ir := lowerTryAbruptIR(t, `try { throw 1; } catch (err) { return err; }`)
	for _, want := range []string{
		"%local.0 = alloca %jayess.value",
		"br label %try.catch.",
		"try.catch.",
		"load %jayess.value, %jayess.value* %local.0",
		"br label %return.",
		"return.",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected try/catch IR to contain %q:\n%s", want, ir)
		}
	}
	if strings.Contains(ir, "jayess_throw_unhandled") {
		t.Fatalf("expected caught throw not to use unhandled throw path:\n%s", ir)
	}
}

func TestLLVMBackendStatementEmitterLowersTryFinallyNormalPath(t *testing.T) {
	ir := lowerTryAbruptIR(t, `var value = 1; try { value = 2; } finally { value = 3; } return value;`)
	for _, want := range []string{
		"br label %try.finally.",
		"try.finally.",
		"br label %try.end.",
		"try.end.",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected try/finally IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterRestoresCatchBindingScope(t *testing.T) {
	ir := lowerTryAbruptIR(t, `var value = 1; try { throw 2; } catch (value) { value = 3; } return value;`)
	for _, want := range []string{
		"try.catch.",
		"%local.2 = alloca %jayess.value",
		"%v4 = load %jayess.value, %jayess.value* %local.0",
		"ret %jayess.value %v4",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected catch binding scope IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerTryAbruptIR(t *testing.T, source string) string {
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
		Name:         "try-abrupt-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
