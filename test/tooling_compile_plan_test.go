package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/llvmbackend"
	"jayess-go/tooling"
)

func TestToolingPlansSharedLibraryCompileFromIR(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("macos-arm64")
	if !ok {
		t.Fatal("expected macos-arm64 target config")
	}
	plan := tooling.PlanCompileFromIR(tooling.CompileRequest{
		Emit:       tooling.EmitShared,
		InputIR:    "define void @init() { ret void }",
		OutputPath: "libapp.dylib",
		Target:     target,
	})
	if !plan.CanBuild() {
		t.Fatalf("expected shared library compile plan to be buildable: %#v", plan)
	}
	if plan.Artifact != llvmbackend.SharedLibArtifact {
		t.Fatalf("expected shared library artifact, got %s", plan.Artifact)
	}
	requireStringSlice(t, plan.SharedLibrary.LinkFlags, []string{"-dynamiclib"})
}

func TestToolingPlansNativeExecutableCompileFromIR(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := tooling.PlanCompileFromIR(tooling.CompileRequest{
		Emit:       tooling.EmitNative,
		InputIR:    "define i32 @main() { ret i32 0 }",
		OutputPath: "app",
		Target:     target,
	})
	if !plan.CanBuild() {
		t.Fatalf("expected native executable compile plan to be buildable: %#v", plan)
	}
	if plan.Artifact != llvmbackend.ExecutableArtifact {
		t.Fatalf("expected executable artifact, got %s", plan.Artifact)
	}
}

func TestToolingCompilePlanIncludesBindingSourceCompilation(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := tooling.PlanCompileFromIR(tooling.CompileRequest{
		Emit:       tooling.EmitShared,
		InputIR:    "define void @init() { ret void }",
		OutputPath: "libmath.so",
		Target:     target,
		BindingModules: []binding.Module{{
			Path: "./native/math.js",
			Manifest: binding.Manifest{
				Sources:     []string{"./math.c"},
				IncludeDirs: []string{"./include"},
				CFlags:      []string{"-DMATH=1"},
				Exports: []binding.Export{
					{Name: "add", Symbol: "math_add", Kind: binding.FunctionExport},
				},
			},
		}},
		BindingPlatform:  "linux",
		RuntimeHeaderDir: "./runtime",
	})

	if !plan.CanBuild() {
		t.Fatalf("expected binding compile plan to be buildable: %#v", plan)
	}
	if len(plan.BindingBuild.CompileUnits) != 1 {
		t.Fatalf("expected one binding compile unit, got %#v", plan.BindingBuild.CompileUnits)
	}
	unit := plan.BindingBuild.CompileUnits[0]
	if unit.ModulePath != "./native/math.js" || unit.Source != "./math.c" {
		t.Fatalf("unexpected compile unit: %#v", unit)
	}
	requireStringSlice(t, unit.IncludeDirs, []string{"native/include", "./runtime"})
	requireStringSlice(t, unit.CFlags, []string{"-DMATH=1"})
}

func TestToolingCompilePlanLinksBindingObjectsAndLibraries(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	module := binding.Module{
		Path: "./native/math.js",
		Manifest: binding.Manifest{
			Sources:         []string{"./math.c", "./extra.c"},
			LibraryDirs:     []string{"./lib"},
			SharedLibraries: []string{"m"},
			LDFlags:         []string{"-ldl"},
			Exports: []binding.Export{
				{Name: "add", Symbol: "math_add", Kind: binding.FunctionExport},
			},
		},
	}

	shared := tooling.PlanCompileFromIR(tooling.CompileRequest{
		Emit:             tooling.EmitShared,
		InputIR:          "define void @init() { ret void }",
		OutputPath:       "libmath.so",
		Target:           target,
		BindingModules:   []binding.Module{module},
		BindingPlatform:  "linux",
		RuntimeHeaderDir: "./runtime",
	})
	requireStringSlice(t, shared.SharedLibrary.ExtraObjectFiles, []string{
		"temp/jayess-bindings/0-math-math.o",
		"temp/jayess-bindings/1-math-extra.o",
	})
	requireStringSlice(t, shared.SharedLibrary.LinkFlags, []string{"-shared", "-Lnative/lib", "-lm", "-ldl"})

	executable := tooling.PlanCompileFromIR(tooling.CompileRequest{
		Emit:             tooling.EmitNative,
		InputIR:          "define i32 @main() { ret i32 0 }",
		OutputPath:       "app",
		Target:           target,
		BindingModules:   []binding.Module{module},
		BindingPlatform:  "linux",
		RuntimeHeaderDir: "./runtime",
	})
	requireStringSlice(t, executable.Executable.ExtraObjectFiles, []string{
		"temp/jayess-bindings/0-math-math.o",
		"temp/jayess-bindings/1-math-extra.o",
	})
	requireStringSlice(t, executable.Executable.LinkFlags, []string{"-Lnative/lib", "-lm", "-ldl"})
}

func TestToolingCompilePlanReportsSharedLibraryMissingInputs(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := tooling.PlanCompileFromIR(tooling.CompileRequest{
		Emit:   tooling.EmitShared,
		Target: target,
	})
	if plan.CanBuild() {
		t.Fatalf("expected missing-input compile plan not to be buildable: %#v", plan)
	}
	if len(plan.Diagnostics) != 2 {
		t.Fatalf("expected missing IR and output diagnostics, got %#v", plan.Diagnostics)
	}
}

func TestToolingDefaultSharedLibraryOutputNameUsesTarget(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("windows-x64")
	if !ok {
		t.Fatal("expected windows target config")
	}
	if got := tooling.DefaultOutputName(tooling.EmitShared, target, "math"); got != "math.dll" {
		t.Fatalf("expected windows shared library name, got %q", got)
	}
}
