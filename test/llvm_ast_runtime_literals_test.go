package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
	jayessruntime "jayess-go/runtime"
)

func TestLLVMBackendLowersParsedPrimitiveRuntimeLiterals(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{source: "undefined", want: "call %jayess.value @jayess_value_undefined()"},
		{source: "null", want: "call %jayess.value @jayess_value_null()"},
		{source: "true", want: "call %jayess.value @jayess_value_from_boolean(i1 1)"},
		{source: "42.5", want: "call %jayess.value @jayess_value_from_number(double 42.5)"},
		{source: `"jayess"`, want: `@.jayess.literal.0 = private unnamed_addr constant [7 x i8] c"jayess\00"`},
		{source: "9007199254740993n", want: `@.jayess.literal.0 = private unnamed_addr constant [17 x i8] c"9007199254740993\00"`},
	}
	for _, tc := range cases {
		expr, err := parser.New(lexer.New(tc.source)).ParseExpression()
		if err != nil {
			t.Fatalf("parse %q: %v", tc.source, err)
		}
		lowered, err := llvmbackend.LowerASTRuntimeLiteral("%value", expr, 0)
		if err != nil {
			t.Fatalf("lower %q: %v", tc.source, err)
		}
		module := llvmbackend.Module{
			Name:         "literal",
			Globals:      lowered.Globals,
			Declarations: lowered.Declarations,
			Functions: []llvmbackend.Function{{
				Name:       "main",
				ReturnType: "i32",
				Body:       append(lowered.Body, "ret i32 0"),
			}},
		}
		if ir := llvmbackend.EmitLLVMIR(module); !strings.Contains(ir, tc.want) {
			t.Fatalf("expected parsed literal %q IR to contain %q:\n%s", tc.source, tc.want, ir)
		}
	}
}

func TestLLVMBackendClassifiesParsedRuntimeLiterals(t *testing.T) {
	cases := []struct {
		source string
		kind   jayessruntime.ValueKind
	}{
		{source: "undefined", kind: jayessruntime.UndefinedValue},
		{source: "null", kind: jayessruntime.NullValue},
		{source: "false", kind: jayessruntime.BooleanValue},
		{source: "7", kind: jayessruntime.NumberValue},
		{source: "7n", kind: jayessruntime.BigIntValue},
		{source: `"x"`, kind: jayessruntime.StringValue},
	}
	for _, tc := range cases {
		expr, err := parser.New(lexer.New(tc.source)).ParseExpression()
		if err != nil {
			t.Fatalf("parse %q: %v", tc.source, err)
		}
		literal, err := llvmbackend.RuntimeLiteralFromAST(expr)
		if err != nil {
			t.Fatalf("classify %q: %v", tc.source, err)
		}
		if literal.Kind != tc.kind {
			t.Fatalf("expected %q to classify as %s, got %s", tc.source, tc.kind, literal.Kind)
		}
	}
}

func TestLLVMBackendRejectsNonLiteralASTRuntimeValue(t *testing.T) {
	expr, err := parser.New(lexer.New("value")).ParseExpression()
	if err != nil {
		t.Fatalf("parse identifier: %v", err)
	}
	if _, err := llvmbackend.RuntimeLiteralFromAST(expr); err == nil {
		t.Fatal("expected identifier to be rejected by primitive literal bridge")
	}
}
