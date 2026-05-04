package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeUtilCapabilities(t *testing.T) {
	for _, name := range []string{"format", "inspect"} {
		if !jayessruntime.HasUtilCapability(name) {
			t.Fatalf("expected util capability %s", name)
		}
	}
}
