package test

import (
	"strings"
	"testing"

	"jayess-go/llvmbackend"
	jayessruntime "jayess-go/runtime"
)

func TestLLVMBackendLowersPrimitiveRuntimeLiterals(t *testing.T) {
	literals := []llvmbackend.RuntimeLiteral{
		{Kind: jayessruntime.UndefinedValue},
		{Kind: jayessruntime.NullValue},
		{Kind: jayessruntime.BooleanValue, Bool: true},
		{Kind: jayessruntime.NumberValue, Number: 42.5},
		{Kind: jayessruntime.StringValue, Text: "jayess"},
		{Kind: jayessruntime.BigIntValue, Text: "9007199254740993"},
	}
	module := llvmbackend.Module{Name: "literals"}
	var body []string
	for index, literal := range literals {
		lowered, err := llvmbackend.LowerRuntimeLiteral("%v"+string(rune('0'+index)), literal, index)
		if err != nil {
			t.Fatalf("lower literal %d: %v", index, err)
		}
		module.Declarations = append(module.Declarations, lowered.Declarations...)
		module.Globals = append(module.Globals, lowered.Globals...)
		body = append(body, lowered.Body...)
	}
	body = append(body, "ret i32 0")
	module.Functions = []llvmbackend.Function{{Name: "main", ReturnType: "i32", Body: body}}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"declare %jayess.value () @jayess_value_undefined",
		"declare %jayess.value () @jayess_value_null",
		"declare %jayess.value (i1) @jayess_value_from_boolean",
		"declare %jayess.value (double) @jayess_value_from_number",
		"declare %jayess.value (i8*) @jayess_value_from_string_copy",
		"declare %jayess.value (i8*) @jayess_value_from_bigint_string",
		"%v2 = call %jayess.value @jayess_value_from_boolean(i1 1)",
		"%v3 = call %jayess.value @jayess_value_from_number(double 42.5)",
		"@.jayess.literal.4 = private unnamed_addr constant [7 x i8] c\"jayess\\00\"",
		"@.jayess.literal.5 = private unnamed_addr constant [17 x i8] c\"9007199254740993\\00\"",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected runtime literal IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendRejectsUnsupportedRuntimeLiteral(t *testing.T) {
	if _, err := llvmbackend.LowerRuntimeLiteral("%v", llvmbackend.RuntimeLiteral{Kind: jayessruntime.ObjectValue}, 0); err == nil {
		t.Fatal("expected object literal runtime lowering to be rejected by primitive literal lowerer")
	}
}
