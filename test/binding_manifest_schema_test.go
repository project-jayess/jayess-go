package test

import (
	"testing"

	"jayess-go/binding"
)

func TestBindingManifestSupportsSchemaAndSymbolMapping(t *testing.T) {
	manifest := binding.Manifest{
		Sources:            []string{"./src/math.c"},
		IncludeDirs:        []string{"./include"},
		LibraryDirs:        []string{"./vendor/lib"},
		SharedLibraries:    []string{"mylib"},
		LicenseFiles:       []string{"./vendor/LICENSE"},
		RuntimeAssets:      []string{"./data/schema.json"},
		HelperAssets:       []string{"./bin/helper"},
		CFlags:             []string{"-DMATH_BINDING=1"},
		LDFlags:            []string{"-lm"},
		PlaceholderExports: []string{"add"},
		Exports: []binding.Export{
			{Name: "add", Symbol: "mylib_add", Kind: binding.FunctionExport},
			{Name: "version", Symbol: "mylib_version", Kind: binding.ValueExport},
		},
	}

	if diagnostics := binding.ValidateManifest(manifest); len(diagnostics) != 0 {
		t.Fatalf("expected valid binding manifest, got %#v", diagnostics)
	}
	export, ok := manifest.ExportByName("add")
	if !ok {
		t.Fatal("expected add export")
	}
	if export.Symbol != "mylib_add" || export.Kind != binding.FunctionExport {
		t.Fatalf("unexpected native symbol mapping: %#v", export)
	}
	value, ok := manifest.ExportByName("version")
	if !ok || value.Kind != binding.ValueExport {
		t.Fatalf("expected value export, got %#v", value)
	}
}

func TestBindingManifestMergesPlatformOverrides(t *testing.T) {
	manifest := binding.Manifest{
		Sources:         []string{"./src/base.c"},
		IncludeDirs:     []string{"./include"},
		LibraryDirs:     []string{"./lib"},
		SharedLibraries: []string{"base"},
		LicenseFiles:    []string{"./LICENSE.base"},
		RuntimeAssets:   []string{"./data/base.dat"},
		HelperAssets:    []string{"./bin/base-helper"},
		CFlags:          []string{"-DBASE=1"},
		LDFlags:         []string{"-lbase"},
		Platforms: map[string]binding.PlatformOptions{
			"linux": {
				Sources:         []string{"./src/linux.c"},
				IncludeDirs:     []string{"./include/linux"},
				LibraryDirs:     []string{"./linux/lib"},
				SharedLibraries: []string{"gtk-3"},
				LicenseFiles:    []string{"./LICENSE.gtk"},
				RuntimeAssets:   []string{"./data/linux.dat"},
				HelperAssets:    []string{"./bin/linux-helper"},
				CFlags:          []string{"-DLINUX=1"},
				LDFlags:         []string{"-ldl"},
			},
		},
		Exports: []binding.Export{{Name: "open", Symbol: "mylib_open", Kind: binding.FunctionExport}},
	}

	inputs := manifest.BuildInputsFor("linux")
	requireStringSlice(t, inputs.Sources, []string{"./src/base.c", "./src/linux.c"})
	requireStringSlice(t, inputs.IncludeDirs, []string{"./include", "./include/linux"})
	requireStringSlice(t, inputs.LibraryDirs, []string{"./lib", "./linux/lib"})
	requireStringSlice(t, inputs.SharedLibraries, []string{"base", "gtk-3"})
	requireStringSlice(t, inputs.LicenseFiles, []string{"./LICENSE.base", "./LICENSE.gtk"})
	requireStringSlice(t, inputs.RuntimeAssets, []string{"./data/base.dat", "./data/linux.dat"})
	requireStringSlice(t, inputs.HelperAssets, []string{"./bin/base-helper", "./bin/linux-helper"})
	requireStringSlice(t, inputs.CFlags, []string{"-DBASE=1", "-DLINUX=1"})
	requireStringSlice(t, inputs.LDFlags, []string{"-lbase", "-ldl"})
}

func TestBindingManifestRejectsMalformedExports(t *testing.T) {
	manifest := binding.Manifest{
		Sources: []string{"./src/math.c"},
		Exports: []binding.Export{
			{Name: "add", Symbol: "mylib_add", Kind: binding.FunctionExport},
			{Name: "add", Symbol: "mylib_add_again", Kind: binding.FunctionExport},
			{Name: "bad", Symbol: "", Kind: "class"},
		},
	}

	diagnostics := binding.ValidateManifest(manifest)
	if len(diagnostics) < 3 {
		t.Fatalf("expected malformed export diagnostics, got %#v", diagnostics)
	}
	requireDiagnostic(t, diagnostics, "duplicate export name")
	requireDiagnostic(t, diagnostics, "export symbol must not be empty")
	requireDiagnostic(t, diagnostics, "export type must be function or value")
}
