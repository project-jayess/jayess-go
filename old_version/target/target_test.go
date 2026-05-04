package target

import "testing"

func TestFromNameSupportedTargets(t *testing.T) {
	for _, spec := range supportedTargets {
		got, err := FromName(spec.name)
		if err != nil {
			t.Fatalf("FromName(%q) returned error: %v", spec.name, err)
		}
		if got != spec.triple {
			t.Fatalf("FromName(%q) = %q, want %q", spec.name, got, spec.triple)
		}
	}
}

func TestFromOSArchSupportedTargets(t *testing.T) {
	for _, spec := range supportedTargets {
		got, err := FromOSArch(spec.goos, spec.goarch)
		if err != nil {
			t.Fatalf("FromOSArch(%q, %q) returned error: %v", spec.goos, spec.goarch, err)
		}
		if got != spec.triple {
			t.Fatalf("FromOSArch(%q, %q) = %q, want %q", spec.goos, spec.goarch, got, spec.triple)
		}
	}
}

func TestUnsupportedTargetErrors(t *testing.T) {
	if _, err := FromName("plan9-x64"); err == nil {
		t.Fatal("FromName returned nil error for unsupported target")
	}
	if _, err := FromOSArch("plan9", "amd64"); err == nil {
		t.Fatal("FromOSArch returned nil error for unsupported host platform")
	}
}
