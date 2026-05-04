package test

import (
	"testing"

	"jayess-go/binding"
)

func TestBindingImportSpecAcceptsNamedExports(t *testing.T) {
	manifest := binding.Manifest{
		Exports: []binding.Export{
			{Name: "add", Symbol: "mylib_add", Kind: binding.FunctionExport},
			{Name: "version", Symbol: "mylib_version", Kind: binding.ValueExport},
		},
	}
	spec := binding.ImportSpec{
		Source: "./native/math.bind.js",
		Kind:   binding.NamedImport,
		Names:  []string{"add", "version"},
	}

	if diagnostics := binding.ValidateImportSpec(spec, manifest); len(diagnostics) != 0 {
		t.Fatalf("expected valid binding import, got %#v", diagnostics)
	}
}

func TestBindingImportSpecRejectsUnsupportedImportForms(t *testing.T) {
	manifest := binding.Manifest{
		Exports: []binding.Export{{Name: "add", Symbol: "mylib_add", Kind: binding.FunctionExport}},
	}
	for _, kind := range []binding.ImportKind{
		binding.DefaultImport,
		binding.NamespaceImport,
		binding.SideEffectImport,
	} {
		spec := binding.ImportSpec{Source: "./native/math.bind.js", Kind: kind, Names: []string{"add"}}
		diagnostics := binding.ValidateImportSpec(spec, manifest)
		requireDiagnostic(t, diagnostics, "binding modules only support named imports")
	}
}

func TestBindingImportSpecReportsMissingExports(t *testing.T) {
	manifest := binding.Manifest{
		Exports: []binding.Export{{Name: "add", Symbol: "mylib_add", Kind: binding.FunctionExport}},
	}
	spec := binding.ImportSpec{
		Source: "./native/math.bind.js",
		Kind:   binding.NamedImport,
		Names:  []string{"missing"},
	}

	diagnostics := binding.ValidateImportSpec(spec, manifest)
	requireDiagnostic(t, diagnostics, "binding export missing was not declared")
}
