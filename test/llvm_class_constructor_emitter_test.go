package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersClassDeclarationAndConstructor(t *testing.T) {
	ir := lowerClassConstructorIR(t, `class Box { constructor() {} } return Box;`)
	for _, want := range []string{
		"@jayess_class_new",
		"@jayess_class_define_constructor",
		"@jayess_function_new",
		"call %jayess.value @jayess_class_new",
		"call void @jayess_class_define_constructor",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected class constructor IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterLowersNewThisAndNewTarget(t *testing.T) {
	ir := lowerClassConstructorIR(t, `class Box { constructor() { this; new.target; } } return new Box(1);`)
	for _, want := range []string{
		"@jayess_class_construct",
		"@jayess_array_push",
		"call %jayess.value @jayess_class_construct",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected new expression IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterLowersThisSuperAndNewTargetPrimitives(t *testing.T) {
	for _, testCase := range []struct {
		source string
		want   string
	}{
		{`return this;`, "@jayess_current_this"},
		{`return new.target;`, "@jayess_new_target"},
		{`return super();`, "@jayess_class_construct_super"},
	} {
		ir := lowerClassConstructorIR(t, testCase.source)
		if !strings.Contains(ir, testCase.want) {
			t.Fatalf("expected %s IR to contain %q:\n%s", testCase.source, testCase.want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterLowersClassExtendsAndSuperConstructor(t *testing.T) {
	ir := lowerClassConstructorIR(t, `class Base {} class Child extends Base { constructor() { super(); } } return new Child();`)
	for _, want := range []string{
		"@jayess_class_extends",
		"@jayess_class_construct",
		"call %jayess.value @jayess_class_extends",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected class extends IR to contain %q:\n%s", want, ir)
		}
	}
}

func lowerClassConstructorIR(t *testing.T, source string) string {
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
		Name:         "class-constructor-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
}
