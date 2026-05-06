package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersBooleanLiteralConditional(t *testing.T) {
	for _, source := range []string{"true ? 11 : 22", "false ? 11 : 22"} {
		t.Run(source, func(t *testing.T) {
			ir := lowerConditionalExpressionIR(t, source)
			for _, want := range []string{
				"call i1 @jayess_value_truthy",
				"br i1 %v1, label %conditional.true.0, label %conditional.false.1",
				"conditional.end.2:",
				"phi %jayess.value",
			} {
				if !strings.Contains(ir, want) {
					t.Fatalf("expected conditional IR to contain %q:\n%s", want, ir)
				}
			}
		})
	}
}

func TestLLVMBackendExpressionEmitterLowersDynamicConditionalCondition(t *testing.T) {
	program, err := parser.New(lexer.New(`var ready = true; return ready ? 1 : 2;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "conditional-statement-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
	for _, want := range []string{"load %jayess.value", "call i1 @jayess_value_truthy", "conditional.end.2:", "phi %jayess.value"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected dynamic conditional IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerConditionalExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("conditional_expr", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "conditional-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
