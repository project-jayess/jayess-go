package test

import (
	"os"
	"path/filepath"
	"testing"

	"jayess-go/binding"
)

func TestBindingAvailabilityReportsMissingHeadersLibrariesAndSymbols(t *testing.T) {
	modulePath := filepath.Join("temp", "binding-availability", testPathName(t), "native", "math.js")
	module := binding.Module{
		Path: modulePath,
		Manifest: binding.Manifest{
			Sources:         []string{"./math.c"},
			IncludeDirs:     []string{"./include"},
			LibraryDirs:     []string{"./lib"},
			SharedLibraries: []string{"./libmissing.so"},
			Exports: []binding.Export{
				{Name: "add", Symbol: "math_add", Kind: binding.FunctionExport},
			},
		},
	}
	plan := binding.PlanBuild([]binding.Module{module}, "linux", filepath.Join(filepath.Dir(modulePath), "runtime"))

	diagnostics := binding.ValidateBuildAvailability(plan, binding.SymbolInventory{
		modulePath: []string{"other_symbol"},
	})

	requireDiagnostic(t, diagnostics, "missing native source")
	requireDiagnostic(t, diagnostics, "missing header directory")
	requireDiagnostic(t, diagnostics, "missing runtime header")
	requireDiagnostic(t, diagnostics, "missing library directory")
	requireDiagnostic(t, diagnostics, "missing shared library")
	requireDiagnostic(t, diagnostics, "missing native symbol math_add")
}

func TestBindingAvailabilityAcceptsPresentFilesAndSymbols(t *testing.T) {
	root := filepath.Join("temp", "binding-availability", testPathName(t))
	modulePath := filepath.Join(root, "native", "math.js")
	nativeDir := filepath.Dir(modulePath)
	includeDir := filepath.Join(nativeDir, "include")
	runtimeDir := filepath.Join(nativeDir, "runtime")
	libDir := filepath.Join(nativeDir, "lib")
	for _, dir := range []string{includeDir, runtimeDir, libDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create dir %s: %v", dir, err)
		}
	}
	writeFixtureFile(t, filepath.Join(nativeDir, "math.c"), "")
	writeFixtureFile(t, filepath.Join(runtimeDir, binding.RuntimeHeader), "")
	writeFixtureFile(t, filepath.Join(libDir, "libmath.so"), "")

	module := binding.Module{
		Path: modulePath,
		Manifest: binding.Manifest{
			Sources:         []string{"./math.c"},
			IncludeDirs:     []string{"./include"},
			LibraryDirs:     []string{"./lib"},
			SharedLibraries: []string{"./lib/libmath.so"},
			Exports: []binding.Export{
				{Name: "add", Symbol: "math_add", Kind: binding.FunctionExport},
			},
		},
	}
	plan := binding.PlanBuild([]binding.Module{module}, "linux", runtimeDir)

	diagnostics := binding.ValidateBuildAvailability(plan, binding.SymbolInventory{
		modulePath: []string{"math_add"},
	})
	if len(diagnostics) != 0 {
		t.Fatalf("expected clean availability diagnostics, got %#v", diagnostics)
	}
}

func writeFixtureFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture file %s: %v", path, err)
	}
}
