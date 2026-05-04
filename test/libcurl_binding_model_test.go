package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/libcurl"
)

func TestLibcurlBindingModuleCanImportBindJS(t *testing.T) {
	module := libcurl.BindingModule{
		Path: "./native/curl.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./curl.c"},
			LDFlags: []string{"-lcurl"},
			Exports: []binding.Export{
				{Name: "createEasy", Symbol: "jayess_curl_create_easy", Kind: binding.FunctionExport},
			},
		},
		APIs:    []libcurl.APIKind{libcurl.EasyAPI, libcurl.TransferAPI},
		Handles: []libcurl.HandleKind{libcurl.EasyHandle, libcurl.HeaderList},
	}

	if diagnostics := libcurl.ValidateBindingModule(module); len(diagnostics) != 0 {
		t.Fatalf("expected valid libcurl binding module, got %#v", diagnostics)
	}
	if !libcurl.SupportsAPI(module, libcurl.TransferAPI) {
		t.Fatal("expected libcurl transfer API support")
	}
}

func TestLibcurlBindingModuleRejectsMalformedTarget(t *testing.T) {
	module := libcurl.BindingModule{
		Path: "./native/curl.c",
		Manifest: binding.Manifest{
			Exports: []binding.Export{{Name: "create", Symbol: "curl_create", Kind: binding.FunctionExport}},
		},
		Handles: []libcurl.HandleKind{libcurl.EasyHandle},
	}

	diagnostics := libcurl.ValidateBindingModule(module)
	requireDiagnostic(t, diagnostics, ".js")
}

func TestLibcurlBuildPlanUsesVendoredSourceWhenRequested(t *testing.T) {
	module := libcurl.BindingModule{
		Path: "./native/curl.bind.js",
		Manifest: binding.Manifest{
			IncludeDirs: []string{"./include"},
			CFlags:      []string{"-DCURL_STATICLIB=1"},
			Exports:     []binding.Export{{Name: "perform", Symbol: "curl_perform", Kind: binding.FunctionExport}},
		},
		Handles:        []libcurl.HandleKind{libcurl.EasyHandle},
		VendoredSource: true,
	}

	plan := libcurl.PlanBuild([]libcurl.BindingModule{module}, "linux", "./runtime")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean libcurl build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 2 {
		t.Fatalf("expected vendored libcurl compile units, got %#v", plan.CompileUnits)
	}
	requireStringSlice(t, plan.CompileUnits[0].IncludeDirs, []string{"native/include", "./runtime"})
	requireStringSlice(t, plan.CompileUnits[0].CFlags, []string{"-DCURL_STATICLIB=1"})
}

func TestLibcurlHandlesRepresentNativeTypesSafely(t *testing.T) {
	for _, kind := range []libcurl.HandleKind{
		libcurl.EasyHandle,
		libcurl.MultiHandle,
		libcurl.HeaderList,
		libcurl.MimeHandle,
	} {
		if !libcurl.SupportsHandle(kind) {
			t.Fatalf("expected libcurl handle support for %s", kind)
		}
	}
}
