package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersClassFieldsMethodsAndAccessors(t *testing.T) {
	ir := lowerClassMemberIR(t, `class Box {
		value = 1;
		method() {}
		get name() {}
		set name(value) {}
	}`)
	for _, want := range []string{
		"@jayess_class_define_field",
		"@jayess_class_define_method",
		"@jayess_class_define_accessor",
		"call void @jayess_class_define_field",
		"call void @jayess_class_define_method",
		"call void @jayess_class_define_accessor",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected class member IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersPrivateAndStaticClassMembers(t *testing.T) {
	ir := lowerClassMemberIR(t, `class Box {
		#secret = 1;
		#read() {}
		static count = 2;
		static {}
	}`)
	for _, want := range []string{
		"@jayess_class_define_private_field",
		"@jayess_class_define_private_method",
		"@jayess_class_define_static_field",
		"@jayess_class_define_static_block",
		"@jayess_class_run_static_blocks",
		"call void @jayess_class_define_private_field",
		"call void @jayess_class_define_private_method",
		"call void @jayess_class_define_static_field",
		"call void @jayess_class_define_static_block",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected private/static class IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersComputedClassMemberKeys(t *testing.T) {
	ir := lowerClassMemberIR(t, `const key = "value"; class Box { [key] = 1; [key]() {} }`)
	if strings.Count(ir, "call void @jayess_class_define_field") != 1 {
		t.Fatalf("expected one computed field definition:\n%s", ir)
	}
	if strings.Count(ir, "call void @jayess_class_define_method") != 1 {
		t.Fatalf("expected one computed method definition:\n%s", ir)
	}
}

func lowerClassMemberIR(t *testing.T, source string) string {
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
		Name:         "class-member-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
