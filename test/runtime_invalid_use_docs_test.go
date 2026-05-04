package test

import (
	"strings"
	"testing"
)

func TestRuntimeInvalidUseRulesAreDocumented(t *testing.T) {
	doc := readRuntimeOwnershipDoc(t)
	required := []string{
		"finalizer can run at most once",
		"preventing double-free",
		"must stay valid until the last retaining owner releases them",
		"Borrowed pointers and views must not be used after the current native call",
		"Freed or closed runtime values must not be reused silently",
		"must report a runtime error or compiler diagnostic",
		"pointer and reference validity across every boundary",
	}

	for _, text := range required {
		if !strings.Contains(doc, text) {
			t.Fatalf("runtime ownership docs missing %q", text)
		}
	}
}
