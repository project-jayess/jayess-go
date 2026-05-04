package test

import (
	"strings"
	"testing"

	"jayess-go/llvmbackend"
)

func TestLLVMPlatformCoverage(t *testing.T) {
	for _, targetName := range []string{"linux-x64", "linux-arm64", "macos-x64", "macos-arm64", "windows-x64"} {
		support, ok := llvmbackend.PlatformSupportFor(targetName)
		if !ok {
			t.Fatalf("expected LLVM platform support for %s", targetName)
		}
		if !support.Executable || !support.CrossObject || !support.ObjectLibraryEmission {
			t.Fatalf("expected executable/object/library support for %#v", support)
		}
	}
}

func TestLLVMPlatformBoundaryDiagnostics(t *testing.T) {
	macos, _ := llvmbackend.PlatformSupportFor("macos-arm64")
	if !strings.Contains(macos.Diagnostic, "Apple SDK") {
		t.Fatalf("expected Apple SDK diagnostic, got %q", macos.Diagnostic)
	}
	windows, _ := llvmbackend.PlatformSupportFor("windows-x64")
	if !strings.Contains(windows.Diagnostic, "Windows SDK") {
		t.Fatalf("expected Windows SDK diagnostic, got %q", windows.Diagnostic)
	}
}
