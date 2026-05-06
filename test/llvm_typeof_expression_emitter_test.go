package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersPrimitiveTypeof(t *testing.T) {
	for _, source := range []string{"typeof undefined", "typeof true", "typeof 42", "typeof 42n", `typeof "jayess"`, "typeof null"} {
		t.Run(source, func(t *testing.T) {
			ir := lowerTypeofExpressionIR(t, source)
			for _, want := range []string{
				"declare %jayess.value (%jayess.value) @jayess_value_typeof",
				"call %jayess.value @jayess_value_typeof",
				"ret %jayess.value",
			} {
				if !strings.Contains(ir, want) {
					t.Fatalf("expected typeof IR to contain %q:\n%s", want, ir)
				}
			}
		})
	}
}

func TestLLVMBackendExpressionEmitterLowersDynamicTypeof(t *testing.T) {
	program, err := parser.New(lexer.New(`var value = 1; return typeof value;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "typeof-statement-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
	for _, want := range []string{
		"load %jayess.value",
		"call %jayess.value @jayess_value_typeof",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected dynamic typeof IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerTypeofExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("typeof_expr", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "typeof-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
