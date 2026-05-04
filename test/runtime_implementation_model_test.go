package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeImplementationModelIsGoFirst(t *testing.T) {
	model := jayessruntime.DefaultImplementationModel()
	if model.RuntimeLanguage != jayessruntime.GoRuntime {
		t.Fatalf("expected Go runtime language, got %s", model.RuntimeLanguage)
	}
	if model.CompilerLanguage != jayessruntime.GoRuntime {
		t.Fatalf("expected Go compiler language, got %s", model.CompilerLanguage)
	}
	if !jayessruntime.RuntimeIsGoFirst(model) {
		t.Fatalf("expected runtime model to be Go-first: %#v", model)
	}
}

func TestRuntimeImplementationKeepsNativeBindingsAtBoundary(t *testing.T) {
	model := jayessruntime.DefaultImplementationModel()
	if model.NativeBindingsAreCore {
		t.Fatal("native bindings must not define the core runtime implementation language")
	}
	if !jayessruntime.SupportsExternalBoundary(model, jayessruntime.CBoundary) {
		t.Fatal("expected optional C boundary for native bindings")
	}
	if !jayessruntime.SupportsExternalBoundary(model, jayessruntime.CPPBoundary) {
		t.Fatal("expected optional C++ boundary for native bindings")
	}
}
