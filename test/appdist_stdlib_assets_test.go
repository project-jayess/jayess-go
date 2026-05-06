package test

import (
	"os"
	"path/filepath"
	"testing"

	"jayess-go/appdist"
	"jayess-go/binding"
)

func TestAppDistIncludesOSCLIRuntimeAssetForStdlibImports(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "runtime", "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatalf("create runtime asset dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "os_cli_runtime.json"), []byte(`{"name":"test"}`), 0o644); err != nil {
		t.Fatalf("write runtime asset: %v", err)
	}
	executable := filepath.Join(root, "build", "demo")
	writeFile(t, executable, "fake exe")

	plan := appdist.PlanApplicationWithStdlibImports(
		executable,
		filepath.Join(root, "dist", "demo"),
		binding.BuildPlan{},
		"linux-x64",
		[]string{"fs", "process", "terminal"},
		assetDir,
	)
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected no appdist diagnostics, got %#v", plan.Diagnostics)
	}
	if len(plan.Assets) != 1 {
		t.Fatalf("expected one OS/CLI runtime asset, got %#v", plan.Assets)
	}
	if plan.Assets[0].OutputName != filepath.Join("runtime", "os_cli_runtime.json") {
		t.Fatalf("unexpected runtime asset output name %q", plan.Assets[0].OutputName)
	}

	result, err := appdist.Create(plan)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(result.OutputDir, "runtime", "os_cli_runtime.json")); err != nil {
		t.Fatalf("expected packaged OS/CLI runtime asset: %v", err)
	}
}

func TestAppDistDoesNotIncludeOSCLIAssetWithoutRelevantImports(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	writeFile(t, executable, "fake exe")

	plan := appdist.PlanApplicationWithStdlibImports(
		executable,
		filepath.Join(root, "dist", "demo"),
		binding.BuildPlan{},
		"linux-x64",
		[]string{"math"},
		filepath.Join(root, "missing"),
	)
	if len(plan.Diagnostics) != 0 || len(plan.Assets) != 0 {
		t.Fatalf("expected no OS/CLI packaging work, got assets=%#v diagnostics=%#v", plan.Assets, plan.Diagnostics)
	}
}
