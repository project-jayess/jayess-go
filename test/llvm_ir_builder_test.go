package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/llvm"
	"jayess-go/llvmc"
)

func TestLLVMPackageBuildsSimpleFunctionIR(t *testing.T) {
	module := llvm.NewModule("main")
	module.SetTargetTriple("x86_64-pc-linux-gnu")
	mainFn, err := module.AddFunction("main", llvm.I32())
	if err != nil {
		t.Fatalf("add function: %v", err)
	}
	entry, err := mainFn.AppendBlock("entry")
	if err != nil {
		t.Fatalf("append block: %v", err)
	}
	if err := entry.Return(llvm.ConstI32(0)); err != nil {
		t.Fatalf("return value: %v", err)
	}

	ir := module.String()
	for _, want := range []string{
		`target triple = "x86_64-pc-linux-gnu"`,
		"define i32 @main() {",
		"entry:",
		"  ret i32 0",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected IR to contain %q, got:\n%s", want, ir)
		}
	}
}

func TestLLVMPackageBuilderEmitsReturns(t *testing.T) {
	module := llvm.NewModule("main")
	mainFn, err := module.AddFunction("main", llvm.I32())
	if err != nil {
		t.Fatalf("add function: %v", err)
	}
	entry, err := mainFn.AppendBlock("entry")
	if err != nil {
		t.Fatalf("append block: %v", err)
	}
	builder := llvm.NewBuilder()
	if err := builder.BuildRet(llvm.ConstI32(0)); err == nil {
		t.Fatal("expected builder without insertion block to fail")
	}
	builder.PositionAtEnd(entry)
	if err := builder.BuildRet(llvm.ConstI32(7)); err != nil {
		t.Fatalf("build return: %v", err)
	}
	if got := module.String(); !strings.Contains(got, "  ret i32 7") {
		t.Fatalf("expected builder return in IR, got:\n%s", got)
	}
}

func TestLLVMPackageRejectsInvalidIRBuilderInputs(t *testing.T) {
	module := llvm.NewModule("main")
	if _, err := module.AddFunction("", llvm.I32()); err == nil {
		t.Fatal("expected empty function name to fail")
	}
	mainFn, err := module.AddFunction("main", llvm.I32())
	if err != nil {
		t.Fatalf("add function: %v", err)
	}
	if _, err := mainFn.AppendBlock(""); err == nil {
		t.Fatal("expected empty block name to fail")
	}
	entry, err := mainFn.AppendBlock("entry")
	if err != nil {
		t.Fatalf("append block: %v", err)
	}
	if err := entry.Return(llvm.ConstI32(1)); err != nil {
		t.Fatalf("first return: %v", err)
	}
	if err := entry.Return(llvm.ConstI32(2)); err == nil {
		t.Fatal("expected second terminator to fail")
	}
}

func TestLLVMPackageBuildsVoidFunctionIR(t *testing.T) {
	module := llvm.NewModule("main")
	mainFn, err := module.AddFunction("main", llvm.Void())
	if err != nil {
		t.Fatalf("add function: %v", err)
	}
	entry, err := mainFn.AppendBlock("entry")
	if err != nil {
		t.Fatalf("append block: %v", err)
	}
	if err := entry.ReturnVoid(); err != nil {
		t.Fatalf("return void: %v", err)
	}
	if got := module.String(); !strings.Contains(got, "  ret void") {
		t.Fatalf("expected void return in IR, got:\n%s", got)
	}
}

func TestLLVMPackageBuilderIRCanEmitObjectWhenBackendAvailable(t *testing.T) {
	if !llvmc.Available() {
		t.Skip("LLVM C API object emitter is not enabled")
	}
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "llvm-builder-object-*")
	output := filepath.Join(dir, "builder.o")
	module := llvm.NewModule("main")
	module.SetTargetTriple("x86_64-pc-linux-gnu")
	mainFn, err := module.AddFunction("main", llvm.I32())
	if err != nil {
		t.Fatalf("add function: %v", err)
	}
	entry, err := mainFn.AppendBlock("entry")
	if err != nil {
		t.Fatalf("append block: %v", err)
	}
	builder := llvm.NewBuilder()
	builder.PositionAtEnd(entry)
	if err := builder.BuildRet(llvm.ConstI32(0)); err != nil {
		t.Fatalf("build return: %v", err)
	}
	if err := llvmc.EmitObject(llvmc.ObjectRequest{
		IR:           module.String(),
		TargetTriple: "x86_64-pc-linux-gnu",
		OutputPath:   output,
	}); err != nil {
		t.Fatalf("emit object from builder IR: %v", err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("stat builder object: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected non-empty builder object")
	}
}
