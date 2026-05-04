package test

import (
	"testing"

	"jayess-go/llvmbackend"
)

func TestLLVMPlansSharedLibraryFromIROutput(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := llvmbackend.PlanSharedLibraryFromIR("define void @init() { ret void }", "libapp.so", target)
	if !plan.CanBuildSharedLibrary() {
		t.Fatalf("expected shared library build plan to be buildable: %#v", plan)
	}
	if len(plan.Steps) != 3 || plan.Steps[0] != llvmbackend.LLVMVerifyStep || plan.Steps[2] != llvmbackend.ClangLinkStep {
		t.Fatalf("unexpected shared library steps: %#v", plan.Steps)
	}
	requireStringSlice(t, plan.LinkFlags, []string{"-shared"})
	if len(plan.ToolchainDiagnostics) == 0 {
		t.Fatal("expected possible toolchain diagnostics to be recorded")
	}
}

func TestLLVMSharedLibraryPlanReportsMissingInputs(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := llvmbackend.PlanSharedLibraryFromIR("", "", target)
	if plan.CanBuildSharedLibrary() {
		t.Fatalf("expected missing-input shared library plan not to be buildable: %#v", plan)
	}
	if len(plan.Diagnostics) != 2 {
		t.Fatalf("expected missing IR and output diagnostics, got %#v", plan.Diagnostics)
	}
}

func TestLLVMSharedLibraryPlanReportsMissingTargetTriple(t *testing.T) {
	plan := llvmbackend.PlanSharedLibraryFromIR("define void @init() { ret void }", "libapp.so", llvmbackend.TargetConfig{})
	if plan.CanBuildSharedLibrary() {
		t.Fatalf("expected missing target triple plan not to be buildable: %#v", plan)
	}
	if len(plan.Diagnostics) != 1 || plan.Diagnostics[0] != "missing LLVM target triple" {
		t.Fatalf("expected target triple diagnostic, got %#v", plan.Diagnostics)
	}
}

func TestLLVMSharedLibraryLinkFlagsFollowTargetPlatform(t *testing.T) {
	cases := map[string][]string{
		"linux-x64":   {"-shared"},
		"macos-arm64": {"-dynamiclib"},
		"windows-x64": {"-shared"},
	}
	for targetName, flags := range cases {
		target, ok := llvmbackend.TargetConfigFor(targetName)
		if !ok {
			t.Fatalf("expected target config for %s", targetName)
		}
		requireStringSlice(t, llvmbackend.SharedLibraryLinkFlags(target), flags)
	}
}

func TestLLVMSharedLibraryNameForTarget(t *testing.T) {
	expected := map[string]string{
		"linux-x64":   "libmath.so",
		"macos-arm64": "libmath.dylib",
		"windows-x64": "math.dll",
	}
	for targetName, want := range expected {
		target, ok := llvmbackend.TargetConfigFor(targetName)
		if !ok {
			t.Fatalf("expected target config for %s", targetName)
		}
		if got := llvmbackend.SharedLibraryNameForTarget(target, "math"); got != want {
			t.Fatalf("shared library name for %s = %q, want %q", targetName, got, want)
		}
	}
}
