package test

import (
	"testing"

	"jayess-go/llvmbackend"
)

func TestLLVMToolchainInteropSteps(t *testing.T) {
	interop := llvmbackend.DefaultToolchainInterop()
	for _, step := range []llvmbackend.ToolchainStep{
		llvmbackend.LLVMVerifyStep,
		llvmbackend.OptStep,
		llvmbackend.LLCStep,
		llvmbackend.ClangLinkStep,
	} {
		if !hasToolchainStep(interop.Steps, step) {
			t.Fatalf("expected LLVM toolchain step %s in %#v", step, interop.Steps)
		}
	}
	if !interop.StableABI {
		t.Fatal("expected stable ABI metadata")
	}
}

func TestLLVMBackendInvariants(t *testing.T) {
	invariants := llvmbackend.BackendInvariants()
	for _, want := range []llvmbackend.BackendInvariant{
		llvmbackend.RuntimeLinkInvariant,
		llvmbackend.NativeBindingLinkInvariant,
		llvmbackend.CCallConventionInvariant,
		llvmbackend.ErrorPathInvariant,
		llvmbackend.DataLayoutInvariant,
	} {
		if !hasBackendInvariant(invariants, want) {
			t.Fatalf("expected backend invariant %s in %#v", want, invariants)
		}
	}
}

func hasToolchainStep(steps []llvmbackend.ToolchainStep, want llvmbackend.ToolchainStep) bool {
	for _, step := range steps {
		if step == want {
			return true
		}
	}
	return false
}

func hasBackendInvariant(invariants []llvmbackend.BackendInvariant, want llvmbackend.BackendInvariant) bool {
	for _, invariant := range invariants {
		if invariant == want {
			return true
		}
	}
	return false
}
