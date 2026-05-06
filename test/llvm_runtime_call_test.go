package test

import (
	"testing"

	"jayess-go/llvmbackend"
)

func TestLLVMBackendRuntimeCallFormatsDeclarationAndCall(t *testing.T) {
	args := []llvmbackend.RuntimeCallArg{
		{IRType: "i1", Value: "1"},
		{IRType: "i8*", Value: "%name"},
	}
	declaration := llvmbackend.RuntimeCallDeclaration("jayess_test_call", "%jayess.value", args)
	if declaration.Name != "jayess_test_call" {
		t.Fatalf("unexpected declaration name %q", declaration.Name)
	}
	if declaration.IRType != "%jayess.value (i1, i8*)" {
		t.Fatalf("unexpected declaration type %q", declaration.IRType)
	}
	call := llvmbackend.RuntimeCall("%v0", "%jayess.value", "jayess_test_call", args)
	want := "%v0 = call %jayess.value @jayess_test_call(i1 1, i8* %name)"
	if call != want {
		t.Fatalf("expected runtime call %q, got %q", want, call)
	}
}

func TestLLVMBackendRuntimeVoidCallFormatsCall(t *testing.T) {
	args := []llvmbackend.RuntimeCallArg{
		{IRType: "%jayess.value", Value: "%object"},
		{IRType: "%jayess.value", Value: "%key"},
	}
	call := llvmbackend.RuntimeVoidCall("jayess_test_store", args)
	want := "call void @jayess_test_store(%jayess.value %object, %jayess.value %key)"
	if call != want {
		t.Fatalf("expected runtime void call %q, got %q", want, call)
	}
}
