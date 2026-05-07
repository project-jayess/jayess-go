package test

import (
	"os"
	"path/filepath"
	"testing"

	"jayess-go/appdist"
	"jayess-go/binding"
)

func TestAppDistIncludesInternalWebviewRuntimeAsset(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "runtime", "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatalf("create runtime asset dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "webview_runtime.json"), []byte(`{"package":"@jayess/webview"}`), 0o644); err != nil {
		t.Fatalf("write runtime asset: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(assetDir, "webview", "windows"), 0o755); err != nil {
		t.Fatalf("create webview backend dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "webview", "windows", "WebView2Loader.dll"), []byte("fake loader"), 0o644); err != nil {
		t.Fatalf("write backend runtime asset: %v", err)
	}
	executable := filepath.Join(root, "build", "demo")
	writeFile(t, executable, "fake exe")

	plan := appdist.PlanApplicationWithStdlibImports(
		executable,
		filepath.Join(root, "dist", "demo"),
		binding.BuildPlan{},
		"windows-x64",
		[]string{"@jayess/webview"},
		assetDir,
	)
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", plan.Diagnostics)
	}
	if !hasRuntimeAsset(plan.Assets, filepath.Join("runtime", "webview_runtime.json")) {
		t.Fatalf("expected webview runtime asset in %#v", plan.Assets)
	}

	result, err := appdist.Create(plan)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	requireFile(t, filepath.Join(result.OutputDir, "runtime", "webview_runtime.json"))
}

func TestAppDistWebviewPackageStaysInternalAcrossSupportedTargets(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "runtime", "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatalf("create runtime asset dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "webview_runtime.json"), []byte(`{"package":"@jayess/webview"}`), 0o644); err != nil {
		t.Fatalf("write runtime asset: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(assetDir, "webview", "windows"), 0o755); err != nil {
		t.Fatalf("create webview backend dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "webview", "windows", "WebView2Loader.dll"), []byte("fake loader"), 0o644); err != nil {
		t.Fatalf("write backend runtime asset: %v", err)
	}
	executable := filepath.Join(root, "build", "demo")
	writeFile(t, executable, "fake exe")

	for _, target := range []string{"linux-x64", "darwin-arm64", "windows-x64"} {
		t.Run(target, func(t *testing.T) {
			plan := appdist.PlanApplicationWithStdlibImports(
				executable,
				filepath.Join(root, "dist", target),
				binding.BuildPlan{},
				target,
				[]string{"@jayess/webview"},
				assetDir,
			)
			if len(plan.Diagnostics) != 0 {
				t.Fatalf("expected no diagnostics, got %#v", plan.Diagnostics)
			}
			for _, asset := range plan.Assets {
				if target != "windows-x64" && (isNativeLibraryAsset(asset.OutputName) || isNativeLibraryAsset(asset.SourcePath)) {
					t.Fatalf("internal webview package should not plan external native library assets: %#v", asset)
				}
			}
			result, err := appdist.Create(plan)
			if err != nil {
				t.Fatalf("Create returned error: %v", err)
			}
			requireFile(t, filepath.Join(result.OutputDir, "demo"))
			requireFile(t, filepath.Join(result.OutputDir, "runtime", "webview_runtime.json"))
		})
	}
}

func hasRuntimeAsset(assets []appdist.RuntimeAsset, outputName string) bool {
	for _, asset := range assets {
		if asset.OutputName == outputName {
			return true
		}
	}
	return false
}
