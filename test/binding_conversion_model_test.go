package test

import (
	"testing"

	"jayess-go/binding"
)

func TestBindingPrimitiveConversionRules(t *testing.T) {
	for _, value := range []binding.ValueKind{
		binding.NumberValue,
		binding.StringValue,
		binding.BooleanValue,
		binding.NullishValue,
	} {
		rule, ok := binding.ConversionRuleFor(value)
		if !ok {
			t.Fatalf("expected conversion rule for %s", value)
		}
		if rule.Native == "" || rule.ToNative == "" || rule.FromNative == "" {
			t.Fatalf("expected complete conversion rule for %s: %#v", value, rule)
		}
		if !rule.TypeChecked {
			t.Fatalf("expected type checked conversion rule for %s", value)
		}
	}
}

func TestBindingReferenceConversionRules(t *testing.T) {
	expectedOwnership := map[binding.ValueKind]binding.OwnershipRule{
		binding.ObjectValue:  binding.BorrowedViewsDuringCall,
		binding.ArrayValue:   binding.BorrowedViewsDuringCall,
		binding.BufferValue:  binding.CopiedBytesForStorage,
		binding.NativeHandle: binding.ManagedHandlesClosable,
	}
	for value, ownership := range expectedOwnership {
		rule, ok := binding.ConversionRuleFor(value)
		if !ok {
			t.Fatalf("expected conversion rule for %s", value)
		}
		if rule.Ownership != ownership {
			t.Fatalf("expected %s ownership for %s, got %s", ownership, value, rule.Ownership)
		}
	}
}

func TestBindingBoundaryErrorKinds(t *testing.T) {
	kinds := binding.BoundaryErrorKinds()
	for _, want := range []binding.BoundaryErrorKind{
		binding.InvalidImportError,
		binding.SymbolResolutionError,
		binding.TypeMismatchError,
		binding.BindingThrownError,
		binding.NativeBuildFailureError,
	} {
		if !hasBoundaryErrorKind(kinds, want) {
			t.Fatalf("expected boundary error kind %s in %#v", want, kinds)
		}
	}
}

func hasBoundaryErrorKind(kinds []binding.BoundaryErrorKind, want binding.BoundaryErrorKind) bool {
	for _, kind := range kinds {
		if kind == want {
			return true
		}
	}
	return false
}
