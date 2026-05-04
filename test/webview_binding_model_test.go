package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/webview"
)

func TestWebviewBindingModuleCanImportBindJS(t *testing.T) {
	module := webview.BindingModule{
		Path: "./native/webview.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./webview.cpp"},
			Exports: []binding.Export{
				{Name: "createWindow", Symbol: "jayess_webview_create_window", Kind: binding.FunctionExport},
			},
		},
		APIs:    []webview.APIKind{webview.WindowAPI, webview.ContentAPI, webview.BridgeAPI},
		Handles: []webview.HandleKind{webview.WebviewHandle, webview.BridgeHandle},
	}

	if diagnostics := webview.ValidateBindingModule(module); len(diagnostics) != 0 {
		t.Fatalf("expected valid webview binding module, got %#v", diagnostics)
	}
	if !webview.SupportsAPI(module, webview.BridgeAPI) {
		t.Fatal("expected webview bridge API support")
	}
}

func TestWebviewBindingModuleRejectsMalformedTarget(t *testing.T) {
	module := webview.BindingModule{
		Path: "./native/webview.cpp",
		Manifest: binding.Manifest{
			Exports: []binding.Export{{Name: "create", Symbol: "webview_create", Kind: binding.FunctionExport}},
		},
		Handles: []webview.HandleKind{webview.WebviewHandle},
	}

	diagnostics := webview.ValidateBindingModule(module)
	requireDiagnostic(t, diagnostics, ".js")
}

func TestWebviewBindingBuildPlanLinksNativeSources(t *testing.T) {
	module := webview.BindingModule{
		Path: "./native/webview.bind.js",
		Manifest: binding.Manifest{
			Sources:     []string{"./webview.cpp"},
			IncludeDirs: []string{"./include"},
			CFlags:      []string{"-DJAYESS_WEBVIEW=1"},
			LDFlags:     []string{"-lgtk-3", "-lwebkit2gtk-4.1"},
			Exports: []binding.Export{
				{Name: "create", Symbol: "jayess_webview_create", Kind: binding.FunctionExport},
			},
		},
		Handles: []webview.HandleKind{webview.WebviewHandle},
	}

	plan := webview.PlanBuild([]webview.BindingModule{module}, "linux", "./runtime")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean webview build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 1 {
		t.Fatalf("expected one webview compile unit, got %#v", plan.CompileUnits)
	}
	requireStringSlice(t, plan.CompileUnits[0].IncludeDirs, []string{"native/include", "./runtime"})
	requireStringSlice(t, plan.CompileUnits[0].CFlags, []string{"-DJAYESS_WEBVIEW=1"})
	requireStringSlice(t, plan.LDFlags, []string{"-lgtk-3", "-lwebkit2gtk-4.1"})
}

func TestWebviewHandleRulesDefineLifecycleOwnership(t *testing.T) {
	for _, kind := range []webview.HandleKind{
		webview.WebviewHandle,
		webview.WindowHandle,
		webview.BridgeHandle,
		webview.ServerHandle,
	} {
		if !webview.SupportsHandle(kind) {
			t.Fatalf("expected webview handle support for %s", kind)
		}
	}
}
