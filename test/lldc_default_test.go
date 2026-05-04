//go:build !jayess_lld

package test

import (
	"strings"
	"testing"

	"jayess-go/lldc"
)

func TestLLDCBackendDefaultsToExternalClang(t *testing.T) {
	if lldc.Available() {
		t.Fatal("expected default test build to use external clang linker")
	}
	if lldc.BackendName() != "external-clang" {
		t.Fatalf("unexpected backend name %q", lldc.BackendName())
	}
	err := lldc.Link(lldc.LinkRequest{
		ObjectPath:   "temp/unused.o",
		OutputPath:   "temp/libunused.so",
		TargetTriple: "x86_64-pc-linux-gnu",
		Shared:       true,
	})
	if err == nil || !strings.Contains(err.Error(), "internal lld linker is not enabled") {
		t.Fatalf("expected disabled lld diagnostic, got %v", err)
	}
}
