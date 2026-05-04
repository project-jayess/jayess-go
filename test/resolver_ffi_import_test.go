package test

import (
	"testing"

	"jayess-go/resolver"
)

func TestResolverRoutesFfiImportToStdlib(t *testing.T) {
	resolved, err := resolver.ResolveImport("/project/src/native/math.js", "ffi")
	if err != nil {
		t.Fatalf("ResolveImport returned error: %v", err)
	}
	if resolved != "jayess:stdlib/ffi" {
		t.Fatalf("expected Jayess ffi stdlib import, got %s", resolved)
	}
}
