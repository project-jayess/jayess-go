package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersBooleanLiteralLogicalExpression(t *testing.T) {
	cases := []struct {
		source string
		branch string
	}{
		{source: "true && 7", branch: "logical.and"},
		{source: "false && 7", branch: "logical.and"},
		{source: "true || 7", branch: "logical.or"},
		{source: "false || 7", branch: "logical.or"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			ir := lowerLogicalExpressionIR(t, tc.source)
			for _, want := range []string{
				"call i1 @jayess_value_truthy",
				"br i1 %v1, label %" + tc.branch + ".true.0, label %" + tc.branch + ".false.1",
				tc.branch + ".end.2:",
				"phi %jayess.value",
			} {
				if !strings.Contains(ir, want) {
					t.Fatalf("expected logical IR to contain %q:\n%s", want, ir)
				}
			}
		})
	}
}

func TestLLVMBackendExpressionEmitterLowersDynamicLogicalLeftOperand(t *testing.T) {
	program, err := parser.New(lexer.New(`var ready = true; return ready && 1;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "logical-statement-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
	for _, want := range []string{"load %jayess.value", "call i1 @jayess_value_truthy", "logical.and.end.2:", "phi %jayess.value"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected dynamic logical IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerLogicalExpressionIR(t *testing.T, source string) string {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("logical_expr", expr)
	if err != nil {
		t.Fatalf("lower expression: %v", err)
	}
	return llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "logical-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
