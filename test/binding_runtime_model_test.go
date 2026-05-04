package test

import (
	"testing"

	"jayess-go/binding"
)

func TestBindingRuntimeHeaderModel(t *testing.T) {
	if binding.RuntimeHeader != "jayess_runtime.h" {
		t.Fatalf("expected low-level runtime header jayess_runtime.h, got %s", binding.RuntimeHeader)
	}
}

func TestBindingOwnershipRules(t *testing.T) {
	rules := binding.OwnershipRules()
	for _, want := range []binding.OwnershipRule{
		binding.RuntimeOwnsReturnedValues,
		binding.BorrowedViewsDuringCall,
		binding.CopiedStringsForStorage,
		binding.CopiedBytesForStorage,
		binding.ManagedHandlesClosable,
		binding.NoDoubleFreeAcrossNative,
		binding.NoUseAfterFreeAcrossNative,
	} {
		if !hasOwnershipRule(rules, want) {
			t.Fatalf("expected ownership rule %s in %#v", want, rules)
		}
	}
}

func hasOwnershipRule(rules []binding.OwnershipRule, want binding.OwnershipRule) bool {
	for _, rule := range rules {
		if rule == want {
			return true
		}
	}
	return false
}
