package test

import (
	"path/filepath"
	"testing"

	"jayess-go/llvm"
	"jayess-go/resolver"
)

func TestLLVMPackageModelExposesCompilerBuildingAPIs(t *testing.T) {
	model := llvm.DefaultPackage()
	for _, api := range []llvm.APIKind{
		llvm.ContextAPI,
		llvm.ModuleAPI,
		llvm.BuilderAPI,
		llvm.TypeAPI,
		llvm.ValueAPI,
		llvm.TargetAPI,
		llvm.ObjectAPI,
		llvm.LinkerAPI,
	} {
		if !llvm.SupportsAPI(model, api) {
			t.Fatalf("expected LLVM package API %s in %#v", api, model.APIs)
		}
	}
}

func TestLLVMPackageModelDeclaresInternalBackends(t *testing.T) {
	model := llvm.DefaultPackage()
	for _, backend := range []llvm.BackendKind{llvm.LLVMCBackend, llvm.LLDBackend} {
		if !llvm.SupportsBackend(model, backend) {
			t.Fatalf("expected LLVM package backend %s in %#v", backend, model.Backends)
		}
	}
}

func TestLLVMPackageModelValidatesDefaultPackage(t *testing.T) {
	if diagnostics := llvm.ValidatePackage(llvm.DefaultPackage()); len(diagnostics) != 0 {
		t.Fatalf("expected valid LLVM package model, got %#v", diagnostics)
	}
}

func TestLLVMPackageModelReportsMissingCoreSurface(t *testing.T) {
	diagnostics := llvm.ValidatePackage(llvm.PackageModel{Import: "llvm", APIs: []llvm.APIKind{llvm.ContextAPI}})
	if len(diagnostics) != 3 {
		t.Fatalf("expected module, object, and backend diagnostics, got %#v", diagnostics)
	}
}

func TestResolverRoutesLLVMPackageAsStdlib(t *testing.T) {
	resolved, err := resolver.ResolveImport(filepath.Join("project", "src", "main.js"), "llvm")
	if err != nil {
		t.Fatalf("ResolveImport returned error: %v", err)
	}
	if resolved != "jayess:stdlib/llvm" {
		t.Fatalf("expected Jayess LLVM stdlib import, got %s", resolved)
	}
}
