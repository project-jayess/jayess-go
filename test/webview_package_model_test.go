package test

import (
	"path/filepath"
	"testing"

	"jayess-go/resolver"
	"jayess-go/webview"
)

func TestWebviewPackageModelExposesFirstPartyJayessImport(t *testing.T) {
	model := webview.DefaultPackage()
	if model.Import != "@jayess/webview" {
		t.Fatalf("unexpected webview package import %q", model.Import)
	}
	if model.RuntimeImport != "jayess-go/runtime/webview" || !model.UsesInternalRuntime {
		t.Fatalf("unexpected webview runtime contract %#v", model)
	}
}

func TestWebviewPackageModelExposesFocusedPublicSurface(t *testing.T) {
	model := webview.DefaultPackage()
	for _, api := range []webview.PublicAPIKind{
		webview.AppLifecycleAPI,
		webview.WindowSurfaceAPI,
		webview.MountSurfaceAPI,
		webview.EventSurfaceAPI,
		webview.DialogSurfaceAPI,
		webview.DropSurfaceAPI,
		webview.RawHostAPI,
	} {
		if !webview.SupportsPublicAPI(model, api) {
			t.Fatalf("expected webview API %s in %#v", api, model.APIs)
		}
	}
}

func TestWebviewPackageModelValidatesDefaultPackage(t *testing.T) {
	if diagnostics := webview.ValidatePackage(webview.DefaultPackage()); len(diagnostics) != 0 {
		t.Fatalf("expected valid webview package model, got %#v", diagnostics)
	}
}

func TestWebviewPackageModelRequiresFocusedPublicAPIs(t *testing.T) {
	diagnostics := webview.ValidatePackage(webview.PackageModel{
		Import:              "@jayess/webview",
		RuntimeImport:       "jayess-go/runtime/webview",
		UsesInternalRuntime: true,
		APIs:                []webview.PublicAPIKind{webview.WindowSurfaceAPI},
	})
	if len(diagnostics) != 5 {
		t.Fatalf("expected missing API diagnostics, got %#v", diagnostics)
	}
}

func TestResolverRoutesWebviewPackageAsStdlib(t *testing.T) {
	resolved, err := resolver.ResolveImport(filepath.Join("project", "src", "main.js"), "@jayess/webview")
	if err != nil {
		t.Fatalf("ResolveImport returned error: %v", err)
	}
	if resolved != "jayess:stdlib/@jayess/webview" {
		t.Fatalf("expected Jayess webview package import, got %s", resolved)
	}
}
