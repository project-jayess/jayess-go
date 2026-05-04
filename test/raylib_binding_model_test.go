package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/raylib"
)

func TestRaylibBindingModuleCanImportBindJS(t *testing.T) {
	module := raylib.BindingModule{
		Path: "./native/raylib.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./raylib_wrapper.c"},
			LDFlags: []string{"-lraylib"},
			Exports: []binding.Export{
				{Name: "initWindow", Symbol: "jayess_raylib_init_window", Kind: binding.FunctionExport},
			},
		},
		APIs:    []raylib.APIKind{raylib.WindowAPI, raylib.RenderAPI},
		Handles: []raylib.HandleKind{raylib.WindowHandle, raylib.TextureHandle},
	}

	if diagnostics := raylib.ValidateBindingModule(module); len(diagnostics) != 0 {
		t.Fatalf("expected valid raylib binding module, got %#v", diagnostics)
	}
	if !raylib.SupportsAPI(module, raylib.RenderAPI) {
		t.Fatal("expected raylib render API support")
	}
}

func TestRaylibBindingModuleRequiresSafeHandles(t *testing.T) {
	module := raylib.BindingModule{
		Path: "./native/raylib.bind.js",
		Manifest: binding.Manifest{
			Exports: []binding.Export{{Name: "init", Symbol: "raylib_init", Kind: binding.FunctionExport}},
		},
		APIs: []raylib.APIKind{raylib.WindowAPI},
	}

	diagnostics := raylib.ValidateBindingModule(module)
	requireDiagnostic(t, diagnostics, "safe handle kinds")
}

func TestRaylibBuildPlanUsesVendoredSourcesWhenRequested(t *testing.T) {
	module := raylib.BindingModule{
		Path: "./native/raylib.bind.js",
		Manifest: binding.Manifest{
			IncludeDirs: []string{"./src"},
			CFlags:      []string{"-DPLATFORM_DESKTOP"},
			LDFlags:     []string{"-lm"},
			Exports:     []binding.Export{{Name: "draw", Symbol: "raylib_draw", Kind: binding.FunctionExport}},
		},
		APIs:           []raylib.APIKind{raylib.WindowAPI, raylib.RenderAPI},
		Handles:        []raylib.HandleKind{raylib.WindowHandle},
		VendoredSource: true,
	}

	plan := raylib.PlanBuild([]raylib.BindingModule{module}, "linux", "./runtime")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean raylib build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 5 {
		t.Fatalf("expected vendored raylib compile units, got %#v", plan.CompileUnits)
	}
	if plan.CompileUnits[0].Source != "./src/rcore.c" {
		t.Fatalf("expected rcore.c first, got %s", plan.CompileUnits[0].Source)
	}
	requireStringSlice(t, plan.CompileUnits[0].IncludeDirs, []string{"native/src", "./runtime"})
	requireStringSlice(t, plan.LDFlags, []string{"-lm"})
}

func TestRaylibCrossPlatformBuildPlanIncludesOverrides(t *testing.T) {
	module := raylib.BindingModule{
		Path: "./native/raylib.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./raylib_wrapper.c"},
			Platforms: map[string]binding.PlatformOptions{
				"windows": {
					LDFlags: []string{"-lopengl32", "-lgdi32", "-lwinmm"},
				},
			},
			Exports: []binding.Export{{Name: "init", Symbol: "raylib_init", Kind: binding.FunctionExport}},
		},
		APIs:    []raylib.APIKind{raylib.WindowAPI},
		Handles: []raylib.HandleKind{raylib.WindowHandle},
	}

	plan := raylib.PlanBuild([]raylib.BindingModule{module}, "windows", "./runtime")
	requireStringSlice(t, plan.LDFlags, []string{"-lopengl32", "-lgdi32", "-lwinmm"})
}
