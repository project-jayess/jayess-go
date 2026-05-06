package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterLowersBooleanNotLiteral(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{source: "!true", want: "%v0 = call %jayess.value @jayess_value_from_boolean(i1 0)"},
		{source: "!false", want: "%v0 = call %jayess.value @jayess_value_from_boolean(i1 1)"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			expr, err := parser.New(lexer.New(tc.source)).ParseExpression()
			if err != nil {
				t.Fatalf("parse expression: %v", err)
			}
			fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("not_expr", expr)
			if err != nil {
				t.Fatalf("lower expression: %v", err)
			}
			ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
				Name:         "boolean-not-module",
				Declarations: declarations,
				Globals:      globals,
				Functions:    []llvmbackend.Function{fn},
			})
			if !strings.Contains(ir, tc.want) {
				t.Fatalf("expected boolean-not IR to contain %q:\n%s", tc.want, ir)
			}
			if !strings.Contains(ir, "ret %jayess.value %v0") {
				t.Fatalf("expected boolean-not IR to return lowered value:\n%s", ir)
			}
		})
	}
}
