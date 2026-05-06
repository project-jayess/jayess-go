package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersLiteralNullishCoalesce(t *testing.T) {
	for _, source := range []string{"null ?? 7", "undefined ?? 8", "false ?? 9", `"ready" ?? 9`} {
		t.Run(source, func(t *testing.T) {
			ir := lowerNullishExpressionIR(t, source)
			for _, want := range []string{
				"call i1 @jayess_value_is_nullish",
				"br i1 %v1, label %nullish.true.0, label %nullish.false.1",
				"nullish.end.2:",
				"phi %jayess.value",
			} {
				if !strings.Contains(ir, want) {
					t.Fatalf("expected nullish IR to contain %q:\n%s", want, ir)
				}
			}
		})
	}
}

func TestLLVMBackendExpressionEmitterLowersDynamicNullishLeftOperand(t *testing.T) {
	program, err := parser.New(lexer.New(`var value = null; return value ?? 1;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "nullish-statement-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
	for _, want := range []string{"load %jayess.value", "call i1 @jayess_value_is_nullish", "nullish.end.2:", "phi %jayess.value"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected dynamic nullish IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerNullishExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("nullish_expr", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "nullish-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
