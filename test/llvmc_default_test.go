//go:build !jayess_llvmc

package test

import (
	"strings"
	"testing"

	"jayess-go/llvmc"
)

func TestLLVMCBackendDefaultsToExternalToolchain(t *testing.T) {
	if llvmc.Available() {
		t.Fatal("expected default test build to use external toolchain backend")
	}
	if llvmc.BackendName() != "external-toolchain" {
		t.Fatalf("unexpected backend name %q", llvmc.BackendName())
	}
	err := llvmc.EmitObject(llvmc.ObjectRequest{
		IR:           "define i32 @main() { ret i32 0 }",
		TargetTriple: "x86_64-pc-linux-gnu",
		OutputPath:   "temp/unused.o",
	})
	if err == nil || !strings.Contains(err.Error(), "LLVM C API object emitter is not enabled") {
		t.Fatalf("expected disabled LLVM C API diagnostic, got %v", err)
	}
}
