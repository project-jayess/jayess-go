package test

import (
	"testing"

	"jayess-go/binding"
)

func TestBindingBuildPlanCompilesAndLinksSources(t *testing.T) {
	modules := []binding.Module{
		{
			Path: "./native/math.bind.js",
			Manifest: binding.Manifest{
				Sources:         []string{"./math.c"},
				IncludeDirs:     []string{"./include"},
				LibraryDirs:     []string{"./vendor/lib"},
				SharedLibraries: []string{"mylib", "./vendor/libhelper.so"},
				LicenseFiles:    []string{"./vendor/LICENSE.helper"},
				CFlags:          []string{"-DMATH=1"},
				LDFlags:         []string{"-lm"},
				Exports:         []binding.Export{{Name: "add", Symbol: "mylib_add", Kind: binding.FunctionExport}},
			},
		},
		{
			Path: "./native/io.bind.js",
			Manifest: binding.Manifest{
				Sources: []string{"./io.c"},
				LDFlags: []string{"-ldl", "-lm"},
				Exports: []binding.Export{{Name: "open", Symbol: "mylib_open", Kind: binding.FunctionExport}},
			},
		},
	}

	plan := binding.PlanBuild(modules, "linux", "./runtime")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 2 {
		t.Fatalf("expected two native compile units, got %#v", plan.CompileUnits)
	}
	requireStringSlice(t, plan.CompileUnits[0].IncludeDirs, []string{"native/include", "./runtime"})
	requireStringSlice(t, plan.CompileUnits[0].CFlags, []string{"-DMATH=1"})
	requireStringSlice(t, plan.LibraryDirs, []string{"native/vendor/lib"})
	requireStringSlice(t, plan.SharedLibraries, []string{"mylib", "./vendor/libhelper.so"})
	requireStringSlice(t, plan.LicenseFiles, []string{"native/vendor/LICENSE.helper"})
	requireStringSlice(t, plan.LDFlags, []string{"-Lnative/vendor/lib", "-lmylib", "native/vendor/libhelper.so", "-lm", "-ldl"})
}

func TestBindingBuildPlanAppliesPlatformCompilationRules(t *testing.T) {
	module := binding.Module{
		Path: "./native/window.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./window.c"},
			Platforms: map[string]binding.PlatformOptions{
				"linux": {
					Sources:         []string{"./window_linux.c"},
					IncludeDirs:     []string{"./linux"},
					LibraryDirs:     []string{"./linux/lib"},
					SharedLibraries: []string{"gtk-3"},
					LicenseFiles:    []string{"./LICENSE.gtk"},
					CFlags:          []string{"-DLINUX=1"},
					LDFlags:         []string{"-lgtk-3"},
				},
			},
			Exports: []binding.Export{{Name: "create", Symbol: "window_create", Kind: binding.FunctionExport}},
		},
	}

	plan := binding.PlanBuild([]binding.Module{module}, "linux", "./runtime")
	if len(plan.CompileUnits) != 2 {
		t.Fatalf("expected base and linux compile units, got %#v", plan.CompileUnits)
	}
	requireStringSlice(t, plan.CompileUnits[1].IncludeDirs, []string{"native/linux", "./runtime"})
	requireStringSlice(t, plan.CompileUnits[1].CFlags, []string{"-DLINUX=1"})
	requireStringSlice(t, plan.LibraryDirs, []string{"native/linux/lib"})
	requireStringSlice(t, plan.SharedLibraries, []string{"gtk-3"})
	requireStringSlice(t, plan.LicenseFiles, []string{"native/LICENSE.gtk"})
	requireStringSlice(t, plan.LDFlags, []string{"-Lnative/linux/lib", "-lgtk-3"})
}

func TestBindingBuildPlanUsesCrossPlatformNativeFlags(t *testing.T) {
	module := binding.Module{
		Path: "./native/webview.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./webview.cpp"},
			Platforms: map[string]binding.PlatformOptions{
				"linux":   {LDFlags: []string{"-lgtk-3", "-lwebkit2gtk-4.1"}},
				"darwin":  {LDFlags: []string{"-framework", "Cocoa", "-framework", "WebKit"}},
				"windows": {LDFlags: []string{"-lole32", "-lcomctl32"}},
			},
			Exports: []binding.Export{{Name: "create", Symbol: "webview_create", Kind: binding.FunctionExport}},
		},
	}

	cases := map[string][]string{
		"linux":   {"-lgtk-3", "-lwebkit2gtk-4.1"},
		"darwin":  {"-framework", "Cocoa", "WebKit"},
		"windows": {"-lole32", "-lcomctl32"},
	}
	for platform, wantFlags := range cases {
		plan := binding.PlanBuild([]binding.Module{module}, platform, "./runtime")
		for _, want := range wantFlags {
			if !hasString(plan.LDFlags, want) {
				t.Fatalf("expected %s ldflag %s in %#v", platform, want, plan.LDFlags)
			}
		}
	}
}

func TestBindingBuildPlanReportsDuplicateNativeSources(t *testing.T) {
	modules := []binding.Module{
		{
			Path: "./native/math.bind.js",
			Manifest: binding.Manifest{
				Sources: []string{"./shared.c"},
				Exports: []binding.Export{{Name: "add", Symbol: "math_add", Kind: binding.FunctionExport}},
			},
		},
		{
			Path: "./native/string.bind.js",
			Manifest: binding.Manifest{
				Sources: []string{"./shared.c"},
				Exports: []binding.Export{{Name: "trim", Symbol: "string_trim", Kind: binding.FunctionExport}},
			},
		},
	}

	plan := binding.PlanBuild(modules, "linux", "./runtime")
	if len(plan.CompileUnits) != 1 {
		t.Fatalf("expected duplicate source to be compiled once, got %#v", plan.CompileUnits)
	}
	requireDiagnostic(t, plan.Diagnostics, "duplicate native source")
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
