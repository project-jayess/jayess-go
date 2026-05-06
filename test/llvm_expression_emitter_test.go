package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionEmitterBuildsPrimitiveLiteralFunction(t *testing.T) {
	expr, err := parser.New(lexer.New(`"jayess"`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeExpressionFunction("literal", expr)
	if err != nil {
		t.Fatalf("lower expression function: %v", err)
	}
	module := llvmbackend.Module{
		Name:         "literal-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"declare %jayess.value (i8*) @jayess_value_from_string_copy",
		`@.jayess.literal.0 = private unnamed_addr constant [7 x i8] c"jayess\00"`,
		"define %jayess.value @literal()",
		"%v0 = call %jayess.value @jayess_value_from_string_copy",
		"ret %jayess.value %v0",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected expression emitter IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendExpressionEmitterDeduplicatesRuntimeDeclarations(t *testing.T) {
	first, err := parser.New(lexer.New("true")).ParseExpression()
	if err != nil {
		t.Fatalf("parse first expression: %v", err)
	}
	second, err := parser.New(lexer.New("false")).ParseExpression()
	if err != nil {
		t.Fatalf("parse second expression: %v", err)
	}
	emitter := llvmbackend.NewExpressionEmitter()
	if _, err := emitter.EmitExpression(first); err != nil {
		t.Fatalf("emit first expression: %v", err)
	}
	if _, err := emitter.EmitExpression(second); err != nil {
		t.Fatalf("emit second expression: %v", err)
	}
	declarations := emitter.Declarations()
	if len(declarations) != 1 {
		t.Fatalf("expected one boolean runtime declaration, got %#v", declarations)
	}
	body := strings.Join(emitter.Body(), "\n")
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_boolean(i1 1)",
		"%v1 = call %jayess.value @jayess_value_from_boolean(i1 0)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected emitted body to contain %q:\n%s", want, body)
		}
	}
}

func TestLLVMBackendExpressionEmitterRejectsUnsupportedExpression(t *testing.T) {
	expr, err := parser.New(lexer.New("name")).ParseExpression()
	if err != nil {
		t.Fatalf("parse expression: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeExpressionFunction("bad", expr); err == nil {
		t.Fatal("expected unsupported expression to be rejected")
	}
}
