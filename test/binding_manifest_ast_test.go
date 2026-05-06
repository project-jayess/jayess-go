package test

import (
	"testing"

	"jayess-go/binding"
)

func TestBindingManifestFromDefaultBindExport(t *testing.T) {
	program := parseProgram(t, `
		import { bind } from "ffi";

		const f = () => {};
		export const add = f;
		export const version = f;

		export default bind({
			sources: ["./src/mylib.c"],
			includeDirs: ["./include"],
			libraryDirs: ["./vendor/lib"],
			sharedLibraries: ["mylib", "./vendor/libhelper.so"],
			licenseFiles: ["./vendor/LICENSE.helper"],
			runtimeAssets: ["./data/schema.json"],
			helperAssets: ["./bin/helper"],
			cflags: ["-DMYLIB=1"],
			ldflags: ["-lm"],
			exports: {
				version: { symbol: "mylib_version", type: "value" },
				add: { symbol: "mylib_add", type: "function" }
			}
		});
	`)

	manifest, diagnostics := binding.ManifestFromProgram(program)
	if len(diagnostics) != 0 {
		t.Fatalf("expected clean manifest extraction, got %#v", diagnostics)
	}
	requireStringSlice(t, manifest.Sources, []string{"./src/mylib.c"})
	requireStringSlice(t, manifest.IncludeDirs, []string{"./include"})
	requireStringSlice(t, manifest.LibraryDirs, []string{"./vendor/lib"})
	requireStringSlice(t, manifest.SharedLibraries, []string{"mylib", "./vendor/libhelper.so"})
	requireStringSlice(t, manifest.LicenseFiles, []string{"./vendor/LICENSE.helper"})
	requireStringSlice(t, manifest.RuntimeAssets, []string{"./data/schema.json"})
	requireStringSlice(t, manifest.HelperAssets, []string{"./bin/helper"})
	requireStringSlice(t, manifest.CFlags, []string{"-DMYLIB=1"})
	requireStringSlice(t, manifest.LDFlags, []string{"-lm"})
	if len(manifest.Exports) != 2 {
		t.Fatalf("expected two exports, got %#v", manifest.Exports)
	}
	if manifest.Exports[0] != (binding.Export{Name: "add", Symbol: "mylib_add", Kind: binding.FunctionExport}) {
		t.Fatalf("unexpected first export: %#v", manifest.Exports[0])
	}
	if manifest.Exports[1] != (binding.Export{Name: "version", Symbol: "mylib_version", Kind: binding.ValueExport}) {
		t.Fatalf("unexpected second export: %#v", manifest.Exports[1])
	}
}

func TestBindingManifestFromDefaultBindExportReadsPlatforms(t *testing.T) {
	program := parseProgram(t, `
		import { bind } from "ffi";

		export default bind({
			sources: ["./base.c"],
			platforms: {
				linux: {
					sources: ["./linux.c"],
					includeDirs: ["./linux"],
					libraryDirs: ["./linux/lib"],
					sharedLibraries: ["gtk-3"],
					licenseFiles: ["./LICENSE.gtk"],
					runtimeAssets: ["./linux/data.dat"],
					helperAssets: ["./linux/helper"],
					cflags: ["-DLINUX=1"],
					ldflags: ["-ldl"]
				}
			},
			exports: {
				open: { symbol: "mylib_open", type: "function" }
			}
		});
	`)

	manifest, diagnostics := binding.ManifestFromProgram(program)
	if len(diagnostics) != 0 {
		t.Fatalf("expected clean manifest extraction, got %#v", diagnostics)
	}
	linux := manifest.Platforms["linux"]
	requireStringSlice(t, linux.Sources, []string{"./linux.c"})
	requireStringSlice(t, linux.IncludeDirs, []string{"./linux"})
	requireStringSlice(t, linux.LibraryDirs, []string{"./linux/lib"})
	requireStringSlice(t, linux.SharedLibraries, []string{"gtk-3"})
	requireStringSlice(t, linux.LicenseFiles, []string{"./LICENSE.gtk"})
	requireStringSlice(t, linux.RuntimeAssets, []string{"./linux/data.dat"})
	requireStringSlice(t, linux.HelperAssets, []string{"./linux/helper"})
	requireStringSlice(t, linux.CFlags, []string{"-DLINUX=1"})
	requireStringSlice(t, linux.LDFlags, []string{"-ldl"})
}

func TestBindingManifestFromProgramReportsMissingBindExport(t *testing.T) {
	program := parseProgram(t, `export default { exports: {} };`)

	_, diagnostics := binding.ManifestFromProgram(program)
	requireDiagnostic(t, diagnostics, "export default bind(...)")
}

func TestBindingManifestFromProgramReportsNonLiteralManifestFields(t *testing.T) {
	program := parseProgram(t, `
		import { bind } from "ffi";
		const sources = ["./native.c"];

		export default bind({
			sources,
			exports: {
				add: { symbol: "mylib_add", type: "function" }
			}
		});
	`)

	_, diagnostics := binding.ManifestFromProgram(program)
	requireDiagnostic(t, diagnostics, "string array literal")
}

func TestBindingManifestFromProgramFeedsValidation(t *testing.T) {
	program := parseProgram(t, `
		import { bind } from "ffi";

		export default bind({
			sources: ["./native.c"],
			exports: {
				add: { symbol: "", type: "function" }
			}
		});
	`)

	manifest, diagnostics := binding.ManifestFromProgram(program)
	if len(diagnostics) != 0 {
		t.Fatalf("expected extraction to succeed before validation, got %#v", diagnostics)
	}
	requireDiagnostic(t, binding.ValidateManifest(manifest), "export symbol must not be empty")
}
