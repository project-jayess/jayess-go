package test

import (
	"testing"

	"jayess-go/binding"
)

func TestBindingSharedPlaceholderExportsCanBeReused(t *testing.T) {
	manifest := binding.Manifest{
		Exports: []binding.Export{
			{Name: "add", Symbol: "mylib_add", Kind: binding.FunctionExport},
			{Name: "subtract", Symbol: "mylib_subtract", Kind: binding.FunctionExport},
		},
	}
	placeholders := []binding.PlaceholderExport{
		{Name: "add", Stub: "f", Kind: binding.SharedPlaceholder},
		{Name: "subtract", Stub: "f", Kind: binding.SharedPlaceholder},
	}

	if diagnostics := binding.ValidatePlaceholderExports(manifest, placeholders); len(diagnostics) != 0 {
		t.Fatalf("expected reusable shared placeholders, got %#v", diagnostics)
	}
}

func TestBindingSharedPlaceholderExportsRequireStubName(t *testing.T) {
	manifest := binding.Manifest{
		Exports: []binding.Export{{Name: "add", Symbol: "mylib_add", Kind: binding.FunctionExport}},
	}
	placeholders := []binding.PlaceholderExport{
		{Name: "add", Kind: binding.SharedPlaceholder},
	}

	diagnostics := binding.ValidatePlaceholderExports(manifest, placeholders)
	requireDiagnostic(t, diagnostics, "shared placeholder export must name its stub binding")
}
