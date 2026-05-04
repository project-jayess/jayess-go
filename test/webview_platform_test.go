package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/webview"
)

func TestWebviewPlatformSupport(t *testing.T) {
	expectedBackend := map[string]string{
		"linux":   "webkit2gtk",
		"darwin":  "wkwebview",
		"windows": "webview2",
	}
	for platform, backend := range expectedBackend {
		support, ok := webview.PlatformSupportFor(platform)
		if !ok {
			t.Fatalf("expected webview platform support for %s", platform)
		}
		if !support.Supported || support.Backend != backend {
			t.Fatalf("expected %s backend for %#v", backend, support)
		}
	}
}

func TestWebviewPlatformSupportReportsMissingToolchain(t *testing.T) {
	support, ok := webview.PlatformSupportFor("plan9")
	if ok {
		t.Fatalf("did not expect webview platform support for %#v", support)
	}
	if support.Diagnostic == "" {
		t.Fatal("expected missing webview platform diagnostic")
	}
}

func TestWebviewCrossPlatformBuildFlags(t *testing.T) {
	module := webview.BindingModule{
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
		Handles: []webview.HandleKind{webview.WebviewHandle},
	}
	cases := map[string][]string{
		"linux":   {"-lgtk-3", "-lwebkit2gtk-4.1"},
		"darwin":  {"-framework", "Cocoa", "WebKit"},
		"windows": {"-lole32", "-lcomctl32"},
	}
	for platform, flags := range cases {
		plan := webview.PlanBuild([]webview.BindingModule{module}, platform, "./runtime")
		for _, flag := range flags {
			if !hasString(plan.LDFlags, flag) {
				t.Fatalf("expected webview %s flag %s in %#v", platform, flag, plan.LDFlags)
			}
		}
	}
}
