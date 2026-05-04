package test

import (
	"testing"

	"jayess-go/llvmbackend"
	"jayess-go/tooling"
)

func TestLLVMBackendArtifactKinds(t *testing.T) {
	for _, kind := range []llvmbackend.ArtifactKind{
		llvmbackend.LLVMIRArtifact,
		llvmbackend.BitcodeArtifact,
		llvmbackend.ObjectArtifact,
		llvmbackend.ExecutableArtifact,
		llvmbackend.StaticLibArtifact,
		llvmbackend.SharedLibArtifact,
	} {
		if !llvmbackend.SupportsArtifact(kind) {
			t.Fatalf("expected LLVM artifact support for %s", kind)
		}
	}
}

func TestLLVMSharedLibraryNamesFollowPlatformConventions(t *testing.T) {
	expected := map[string]string{
		"linux":   "libmath.so",
		"darwin":  "libmath.dylib",
		"windows": "math.dll",
	}
	for platform, want := range expected {
		if got := llvmbackend.SharedLibraryName(platform, "math"); got != want {
			t.Fatalf("shared library name for %s = %q, want %q", platform, got, want)
		}
	}
}

func TestToolingEmitKindsIncludeLLVMArtifacts(t *testing.T) {
	for _, kind := range []tooling.EmitKind{
		tooling.EmitLLVMIR,
		tooling.EmitBitcode,
		tooling.EmitObject,
		tooling.EmitStatic,
		tooling.EmitShared,
		tooling.EmitNative,
	} {
		if !tooling.HasEmitKind(kind) {
			t.Fatalf("expected tooling emit kind %s", kind)
		}
	}
}
