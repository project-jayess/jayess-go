package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimePackagesAreGoModeled(t *testing.T) {
	packages := jayessruntime.GoRuntimePackages()
	if len(packages) == 0 {
		t.Fatal("expected Go runtime packages")
	}
	for _, pkg := range packages {
		if pkg.Import == "" {
			t.Fatalf("runtime package %s has empty Go import path", pkg.Name)
		}
		if pkg.Language != jayessruntime.GoRuntime {
			t.Fatalf("runtime package %s must be Go, got %s", pkg.Name, pkg.Language)
		}
	}
}

func TestCoreRuntimePackagesStayGoFirst(t *testing.T) {
	if !jayessruntime.CoreRuntimePackagesAreGo(jayessruntime.GoRuntimePackages()) {
		t.Fatal("core runtime packages must stay Go-first")
	}
	if !jayessruntime.HasGoRuntimePackage("mvp-globals") {
		t.Fatal("expected mvp-globals Go runtime package")
	}
	if jayessruntime.HasGoRuntimePackage("native-c-runtime") {
		t.Fatal("did not expect native C runtime package in core runtime model")
	}
}

func TestRuntimePackageModelValidatesDefaultPackages(t *testing.T) {
	diagnostics := jayessruntime.ValidateGoRuntimePackages(jayessruntime.GoRuntimePackages())
	if len(diagnostics) != 0 {
		t.Fatalf("expected valid default runtime packages, got %#v", diagnostics)
	}
}

func TestRuntimePackageModelReportsInvalidPackages(t *testing.T) {
	diagnostics := jayessruntime.ValidateGoRuntimePackages([]jayessruntime.PackageModel{
		{Name: "core", Import: "jayess-go/runtime", Role: jayessruntime.CoreRuntimeRole, Language: jayessruntime.GoRuntime},
		{Name: "core", Import: "", Language: jayessruntime.ImplementationLanguage("c")},
		{},
	})
	if len(diagnostics) != 5 {
		t.Fatalf("expected duplicate, import, role, language, and empty-name diagnostics, got %#v", diagnostics)
	}
}

func TestRuntimePackageModelRequiresCoreRuntimePackage(t *testing.T) {
	diagnostics := jayessruntime.ValidateGoRuntimePackages([]jayessruntime.PackageModel{
		{Name: "filesystem", Import: "jayess-go/runtime", Role: jayessruntime.StdlibRole, Language: jayessruntime.GoRuntime},
	})
	if len(diagnostics) != 1 || diagnostics[0].Message != "runtime package registry must include a core runtime package" {
		t.Fatalf("expected missing core runtime diagnostic, got %#v", diagnostics)
	}
}

func TestRuntimePackageModelRejectsUnknownRole(t *testing.T) {
	diagnostics := jayessruntime.ValidateGoRuntimePackages([]jayessruntime.PackageModel{
		{Name: "mvp-globals", Import: "jayess-go/runtime", Role: jayessruntime.CoreRuntimeRole, Language: jayessruntime.GoRuntime},
		{Name: "plugin", Import: "jayess-go/runtime", Role: jayessruntime.PackageRole("plugin"), Language: jayessruntime.GoRuntime},
	})
	if len(diagnostics) != 1 || diagnostics[0].Message != "unknown runtime package role" {
		t.Fatalf("expected unknown role diagnostic, got %#v", diagnostics)
	}
}
