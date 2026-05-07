package test

import (
	"os"
	"path/filepath"
	"testing"

	"jayess-go/appdist"
	"jayess-go/binding"
	runtimewebview "jayess-go/runtime/webview"
)

func TestAppDistWebviewWindowsPackagesBackendRuntimeWhenProvided(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "runtime", "assets")
	if err := os.MkdirAll(filepath.Join(assetDir, "webview", "windows"), 0o755); err != nil {
		t.Fatalf("create runtime asset dir: %v", err)
	}
	writeFile(t, filepath.Join(assetDir, "webview_runtime.json"), `{"package":"@jayess/webview"}`)
	writeFile(t, filepath.Join(assetDir, "webview", "windows", "WebView2Loader.dll"), "fake loader")
	executable := filepath.Join(root, "build", "demo.exe")
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
		t.Fatalf("unexpected diagnostics %#v", plan.Diagnostics)
	}
	requireRuntimeAssetOutput(t, plan.Assets, filepath.Join("runtime", "webview_runtime.json"))
	requireRuntimeAssetOutput(t, plan.Assets, filepath.Join("webview", "windows", "WebView2Loader.dll"))
}

func TestAppDistWebviewAppleAndLinuxUseDocumentedSystemPrerequisites(t *testing.T) {
	for _, target := range []string{"darwin-arm64", "linux-x64"} {
		t.Run(target, func(t *testing.T) {
			requirement, ok := runtimewebview.BackendRequirementForTarget(target)
			if !ok {
				t.Fatalf("expected backend requirement for %s", target)
			}
			if len(requirement.SystemPrerequisites) == 0 {
				t.Fatalf("expected documented system prerequisites for %s", target)
			}
		})
	}
}

func TestAppDistWebviewDoesNotNeedSeparateJayessPackageInstall(t *testing.T) {
	support := runtimewebview.DefaultSupport()
	if support.RequiresPackageInstall || support.RequiresEndUserInstall {
		t.Fatalf("unexpected install requirement flags %#v", support)
	}
}

func requireRuntimeAssetOutput(t *testing.T, assets []appdist.RuntimeAsset, outputName string) {
	t.Helper()
	for _, asset := range assets {
		if asset.OutputName == outputName {
			return
		}
	}
	t.Fatalf("expected runtime asset output %q in %#v", outputName, assets)
}
