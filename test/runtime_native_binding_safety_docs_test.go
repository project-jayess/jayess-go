package test

import (
	"strings"
	"testing"
)

func TestRuntimeNativeBindingSafetyRulesAreDocumented(t *testing.T) {
	doc := readRuntimeOwnershipDoc(t)
	required := []string{
		"Native wrappers must not store borrowed Jayess pointers beyond the current call",
		"must copy strings or bytes with the runtime copy helpers",
		"Managed native handles become invalid after close",
		"Repeated close on a managed native handle is safe",
		"Using a closed managed native handle must report a runtime error",
		"Native finalizers must run at most once",
	}

	for _, text := range required {
		if !strings.Contains(doc, text) {
			t.Fatalf("runtime ownership docs missing %q", text)
		}
	}
}
