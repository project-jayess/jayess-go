package test

import (
	"strings"
	"testing"

	"jayess-go/llvmbackend"
)

func TestLLVMIRTextEmission(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected target config")
	}
	ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:   "hello",
		Target: target,
		Functions: []llvmbackend.Function{
			{Name: "main", ReturnType: "i32", Body: []string{"ret i32 0"}},
		},
	})

	for _, want := range []string{
		"; ModuleID = 'hello'",
		`target triple = "x86_64-pc-linux-gnu"`,
		"define i32 @main()",
		"ret i32 0",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected emitted IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMIRDeclarationEmission(t *testing.T) {
	ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name: "runtime",
		Declarations: []llvmbackend.Declaration{
			{Name: "jayess_runtime_init", IRType: "void ()"},
		},
	})
	if !strings.Contains(ir, "declare void () @jayess_runtime_init") {
		t.Fatalf("expected runtime declaration in IR:\n%s", ir)
	}
}
