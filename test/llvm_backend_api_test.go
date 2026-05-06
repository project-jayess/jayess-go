package test

import (
	"strings"
	"testing"

	"jayess-go/llvm"
)

func TestLLVMBackendAPIBuildsSmallModule(t *testing.T) {
	module := llvm.NewModule("small")
	module.SetTargetTriple("x86_64-unknown-linux-gnu")
	function, err := module.AddFunction("main", llvm.I32())
	if err != nil {
		t.Fatal(err)
	}
	block, err := function.AppendBlock("entry")
	if err != nil {
		t.Fatal(err)
	}
	builder := llvm.NewBuilder()
	builder.PositionAtEnd(block)
	if err := builder.BuildRet(llvm.ConstI32(0)); err != nil {
		t.Fatal(err)
	}
	ir := module.String()
	if !strings.Contains(ir, `target triple = "x86_64-unknown-linux-gnu"`) {
		t.Fatalf("expected target triple in IR, got:\n%s", ir)
	}
	if !strings.Contains(ir, "define i32 @main()") || !strings.Contains(ir, "ret i32 0") {
		t.Fatalf("expected small function IR, got:\n%s", ir)
	}
}

func TestLLVMBackendAPIValidatesObjectEmissionRequest(t *testing.T) {
	err := llvm.EmitObject(llvm.ObjectEmissionRequest{})
	if err == nil || !strings.Contains(err.Error(), "requires a module") {
		t.Fatalf("expected missing module error, got %v", err)
	}
}

func TestLLVMBackendAPIDeclaresObjectAndLinkAvailability(t *testing.T) {
	_ = llvm.ObjectEmitterAvailable()
	_ = llvm.LinkerAvailable()
}
