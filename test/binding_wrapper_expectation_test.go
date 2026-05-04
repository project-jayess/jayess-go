package test

import (
	"testing"

	"jayess-go/binding"
)

func TestBindingWrapperExpectationsDeriveStableWrapperSymbols(t *testing.T) {
	manifest := binding.Manifest{
		Exports: []binding.Export{
			{Name: "add", Symbol: "mylib_add", Kind: binding.FunctionExport},
			{Name: "kebab-name", Symbol: "mylib_kebab", Kind: binding.ValueExport},
		},
	}

	expectations := binding.WrapperExpectations(manifest)
	if len(expectations) != 2 {
		t.Fatalf("expected two wrapper expectations, got %#v", expectations)
	}
	if expectations[0].WrapperSymbol != "jayess_binding_export_add" {
		t.Fatalf("unexpected add wrapper symbol: %#v", expectations[0])
	}
	if expectations[1].WrapperSymbol != "jayess_binding_export_kebab_name" {
		t.Fatalf("unexpected kebab wrapper symbol: %#v", expectations[1])
	}
}

func TestBindingValidateExportedSymbolsForGeneratedWrappers(t *testing.T) {
	manifest := binding.Manifest{
		Exports: []binding.Export{
			{Name: "bad", Symbol: "not-a-c-symbol", Kind: binding.FunctionExport},
			{Name: "runtime", Symbol: "jayess_value_to_number", Kind: binding.FunctionExport},
			{Name: "first", Symbol: "duplicate_symbol", Kind: binding.ValueExport},
			{Name: "second", Symbol: "duplicate_symbol", Kind: binding.ValueExport},
			{Name: "add", Symbol: "jayess_binding_export_add", Kind: binding.FunctionExport},
		},
	}

	diagnostics := binding.ValidateExportedSymbols(manifest)
	requireDiagnostic(t, diagnostics, "export symbol must be a valid C symbol")
	requireDiagnostic(t, diagnostics, "export symbol must not collide with Jayess runtime header functions")
	requireDiagnostic(t, diagnostics, "export symbol duplicates native symbol used by first")
	requireDiagnostic(t, diagnostics, "export symbol must not collide with generated wrapper symbol")
}
