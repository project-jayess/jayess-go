package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/picohttpparser"
)

func TestPicoHTTPParserBindingModuleCanImportBindJS(t *testing.T) {
	module := picohttpparser.BindingModule{
		Path: "./native/picohttpparser.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./picohttpparser_wrapper.c"},
			Exports: []binding.Export{
				{Name: "parseRequest", Symbol: "jayess_pico_parse_request", Kind: binding.FunctionExport},
			},
		},
		APIs: []picohttpparser.APIKind{
			picohttpparser.RequestAPI,
			picohttpparser.HeadersAPI,
		},
	}

	if diagnostics := picohttpparser.ValidateBindingModule(module); len(diagnostics) != 0 {
		t.Fatalf("expected valid picohttpparser binding module, got %#v", diagnostics)
	}
	if !picohttpparser.SupportsAPI(module, picohttpparser.RequestAPI) {
		t.Fatal("expected request parser API support")
	}
}

func TestPicoHTTPParserBindingModuleRejectsMissingAPIs(t *testing.T) {
	module := picohttpparser.BindingModule{
		Path: "./native/picohttpparser.bind.js",
		Manifest: binding.Manifest{
			Exports: []binding.Export{{Name: "parse", Symbol: "pico_parse", Kind: binding.FunctionExport}},
		},
	}

	diagnostics := picohttpparser.ValidateBindingModule(module)
	requireDiagnostic(t, diagnostics, "supported parser APIs")
}

func TestPicoHTTPParserBuildPlanUsesVendoredSourceWhenRequested(t *testing.T) {
	module := picohttpparser.BindingModule{
		Path: "./native/picohttpparser.bind.js",
		Manifest: binding.Manifest{
			IncludeDirs: []string{"./include"},
			CFlags:      []string{"-DPICOHTTPPARSER_USE_FAST_PATH=1"},
			Exports:     []binding.Export{{Name: "parse", Symbol: "pico_parse", Kind: binding.FunctionExport}},
		},
		APIs:           []picohttpparser.APIKind{picohttpparser.RequestAPI},
		VendoredSource: true,
	}

	plan := picohttpparser.PlanBuild([]picohttpparser.BindingModule{module}, "linux", "./runtime")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean picohttpparser build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 1 {
		t.Fatalf("expected vendored picohttpparser compile unit, got %#v", plan.CompileUnits)
	}
	if plan.CompileUnits[0].Source != "./picohttpparser.c" {
		t.Fatalf("expected vendored picohttpparser.c source, got %s", plan.CompileUnits[0].Source)
	}
	requireStringSlice(t, plan.CompileUnits[0].IncludeDirs, []string{"native/include", "./runtime"})
}

func TestPicoHTTPParserDiagnosticKindsIncludeMissingInputs(t *testing.T) {
	kinds := picohttpparser.DiagnosticKinds()
	for _, want := range []picohttpparser.DiagnosticKind{
		picohttpparser.MissingHeaders,
		picohttpparser.MissingSource,
		picohttpparser.MalformedInput,
		picohttpparser.IncompleteData,
	} {
		if !picoHTTPParserHasDiagnosticKind(kinds, want) {
			t.Fatalf("expected diagnostic kind %s in %#v", want, kinds)
		}
	}
}

func picoHTTPParserHasDiagnosticKind(values []picohttpparser.DiagnosticKind, want picohttpparser.DiagnosticKind) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
