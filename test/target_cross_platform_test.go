package test

import (
	"testing"

	"jayess-go/target"
)

func TestTargetSupportedPlatforms(t *testing.T) {
	for _, name := range []string{"linux-x64", "linux-arm64", "macos-x64", "macos-arm64", "windows-x64"} {
		if _, ok := target.Lookup(name); !ok {
			t.Fatalf("expected target %s", name)
		}
	}
}

func TestTargetTriples(t *testing.T) {
	expected := map[string]string{
		"linux-x64":    "x86_64-pc-linux-gnu",
		"linux-arm64":  "aarch64-unknown-linux-gnu",
		"macos-x64":    "x86_64-apple-darwin",
		"macos-arm64":  "arm64-apple-darwin",
		"windows-x64":  "x86_64-pc-windows-msvc",
		"darwin-arm64": "arm64-apple-darwin",
	}
	for name, want := range expected {
		got, ok := target.Triple(name)
		if !ok {
			t.Fatalf("expected target triple for %s", name)
		}
		if got != want {
			t.Fatalf("target triple for %s = %q, want %q", name, got, want)
		}
	}
}

func TestTargetRuntimeAndOSBehaviorMetadata(t *testing.T) {
	for _, spec := range target.Supported() {
		if spec.RuntimeLinkage == "" {
			t.Fatalf("target %s missing runtime linkage", spec.Name)
		}
		if spec.PathSeparator == "" {
			t.Fatalf("target %s missing path separator", spec.Name)
		}
		if spec.PermissionMode == "" {
			t.Fatalf("target %s missing permission mode", spec.Name)
		}
		if !spec.Networking {
			t.Fatalf("target %s should declare networking behavior support", spec.Name)
		}
	}
}

func TestTargetLookupOSArch(t *testing.T) {
	spec, ok := target.LookupOSArch("linux", "amd64")
	if !ok {
		t.Fatal("expected linux/amd64 target")
	}
	if spec.Name != "linux-x64" {
		t.Fatalf("expected linux-x64 target, got %s", spec.Name)
	}
}
