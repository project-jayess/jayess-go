package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendStatementEmitterLowersSimpleAssignmentReturn(t *testing.T) {
	program, err := parser.New(lexer.New(`var answer = 1; answer = 2; return answer;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
	if err != nil {
		t.Fatalf("lower statements: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "assignment-module",
		Declarations: declarations,
		Globals:      globals,
		Functions:    []llvmbackend.Function{fn},
	})
	for _, want := range []string{
		"%v0 = call %jayess.value @jayess_value_from_number(double 1)",
		"%local.0 = alloca %jayess.value",
		"store %jayess.value %v0, %jayess.value* %local.0",
		"%v1 = call %jayess.value @jayess_value_from_number(double 2)",
		"store %jayess.value %v1, %jayess.value* %local.0",
		"%v2 = load %jayess.value, %jayess.value* %local.0",
		"ret %jayess.value %v2",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected assignment IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMBackendStatementEmitterRejectsUndefinedAssignmentTarget(t *testing.T) {
	program, err := parser.New(lexer.New(`missing = 2;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements); err == nil {
		t.Fatal("expected assignment to undefined emitted local to be rejected")
	}
}

func TestLLVMBackendStatementEmitterLowersRuntimeAssignmentTargets(t *testing.T) {
	cases := []struct {
		name string
		code string
		want string
	}{
		{name: "member", code: `var value = 1; value.name = 2;`, want: "call void @jayess_value_set_property"},
		{name: "index", code: `var values = 1; values[0] = 2;`, want: "call void @jayess_value_set_index"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program, err := parser.New(lexer.New(tc.code)).ParseProgram()
			if err != nil {
				t.Fatalf("parse program: %v", err)
			}
			fn, declarations, globals, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements)
			if err != nil {
				t.Fatalf("lower statements: %v", err)
			}
			ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
				Name:         "assignment-target-module",
				Declarations: declarations,
				Globals:      globals,
				Functions:    []llvmbackend.Function{fn},
			})
			if !strings.Contains(ir, tc.want) {
				t.Fatalf("expected assignment target IR to contain %q:\n%s", tc.want, ir)
			}
		})
	}
}

func TestLLVMBackendStatementEmitterRejectsCompoundAssignment(t *testing.T) {
	program, err := parser.New(lexer.New(`var answer = 1; answer += 2;`)).ParseProgram()
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	if _, _, _, err := llvmbackend.LowerRuntimeStatementFunction("main", program.Statements); err == nil {
		t.Fatal("expected compound assignment to be rejected")
	}
}
