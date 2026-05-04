package test

import (
	"testing"

	"jayess-go/typesys"
)

func TestTypeSystemDefaultPolicy(t *testing.T) {
	policy := typesys.DefaultPolicy()
	if !policy.OptionalOnly {
		t.Fatal("expected optional typing policy")
	}
	if !policy.ErasedAtCompile {
		t.Fatal("expected erased-at-compile policy")
	}
	if !policy.TypedUntypedInterop {
		t.Fatal("expected typed/untyped interop policy")
	}
	if policy.CastSyntax != "assertion" {
		t.Fatalf("expected assertion cast syntax policy, got %q", policy.CastSyntax)
	}
	if policy.RuntimeChecks != typesys.RuntimeChecksUnsupported {
		t.Fatalf("expected unsupported runtime checks policy, got %s", policy.RuntimeChecks)
	}
}

func TestTypeSystemPolicyHelpers(t *testing.T) {
	policy := typesys.DefaultPolicy()
	if !typesys.SupportsTypedUntypedInterop(policy) {
		t.Fatal("expected typed/untyped interop helper to accept default policy")
	}
	if !typesys.ErasesTypes(policy) {
		t.Fatal("expected erase helper to accept default policy")
	}

	policy.OptionalOnly = false
	if typesys.SupportsTypedUntypedInterop(policy) {
		t.Fatal("did not expect interop helper without optional typing")
	}
	if typesys.ErasesTypes(policy) {
		t.Fatal("did not expect erase helper without optional typing")
	}
}

func TestTypeSystemRuntimeCheckPolicyValues(t *testing.T) {
	values := []typesys.RuntimeCheckPolicy{
		typesys.RuntimeChecksUnsupported,
		typesys.RuntimeChecksOptional,
		typesys.RuntimeChecksRequired,
	}
	for _, value := range values {
		if value == "" {
			t.Fatalf("runtime check policy value must not be empty")
		}
	}
}
