package test

import (
	"os"
	"path/filepath"
	"strings"
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

func TestAppDistDoesNotPackageExternalAssetForInternalCrypto(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	writeFile(t, executable, "fake exe")

	plan := appdist.PlanApplicationWithStdlibImports(
		executable,
		filepath.Join(root, "dist", "demo"),
		binding.BuildPlan{},
		"linux-x64",
		[]string{"crypto"},
		filepath.Join(root, "missing"),
	)
	if len(plan.Diagnostics) != 0 || len(plan.Assets) != 0 {
		t.Fatalf("expected internal crypto to need no external assets, got assets=%#v diagnostics=%#v", plan.Assets, plan.Diagnostics)
	}
}

func TestAppDistDoesNotPackageExternalAssetForInternalTLSHTTPS(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	writeFile(t, executable, "fake exe")

	plan := appdist.PlanApplicationWithStdlibImports(
		executable,
		filepath.Join(root, "dist", "demo"),
		binding.BuildPlan{},
		"linux-x64",
		[]string{"tls", "https"},
		filepath.Join(root, "missing"),
	)
	if len(plan.Diagnostics) != 0 || len(plan.Assets) != 0 {
		t.Fatalf("expected internal TLS/HTTPS to need no external assets, got assets=%#v diagnostics=%#v", plan.Assets, plan.Diagnostics)
	}
}

func TestAppDistDoesNotPackageNativeLibrariesForInternalNodeLikeImports(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "runtime", "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatalf("create runtime asset dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "os_cli_runtime.json"), []byte(`{"name":"os-cli"}`), 0o644); err != nil {
		t.Fatalf("write runtime asset: %v", err)
	}
	executable := filepath.Join(root, "build", "demo")
	writeFile(t, executable, "fake exe")

	plan := appdist.PlanApplicationWithStdlibImports(
		executable,
		filepath.Join(root, "dist", "demo"),
		binding.BuildPlan{},
		"linux-x64",
		[]string{
			"Buffer",
			"compression",
			"crypto",
			"dns",
			"http",
			"https",
			"stream",
			"tcp",
			"tls",
			"udp",
			"url",
			"util",
			"worker",
		},
		assetDir,
	)
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics for internal imports, got %#v", plan.Diagnostics)
	}
	for _, asset := range plan.Assets {
		if isNativeLibraryAsset(asset.OutputName) || isNativeLibraryAsset(asset.SourcePath) {
			t.Fatalf("internal import planned native library asset: %#v", asset)
		}
	}
}

func TestAppDistHTTPHTTPSDoNotPackageExternalNetworkingAssets(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "runtime", "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatalf("create runtime asset dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "os_cli_runtime.json"), []byte(`{"name":"os-cli"}`), 0o644); err != nil {
		t.Fatalf("write runtime asset: %v", err)
	}
	executable := filepath.Join(root, "build", "demo")
	writeFile(t, executable, "fake exe")

	plan := appdist.PlanApplicationWithStdlibImports(
		executable,
		filepath.Join(root, "dist", "demo"),
		binding.BuildPlan{},
		"linux-x64",
		[]string{"http", "https", "dns", "tcp", "udp", "stream"},
		assetDir,
	)
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics for internal networking imports, got %#v", plan.Diagnostics)
	}
	for _, asset := range plan.Assets {
		if isNativeLibraryAsset(asset.OutputName) || isNativeLibraryAsset(asset.SourcePath) {
			t.Fatalf("internal networking import planned native library asset: %#v", asset)
		}
		if containsExternalNetworkingAsset(asset.OutputName) || containsExternalNetworkingAsset(asset.SourcePath) {
			t.Fatalf("internal networking import planned external networking asset: %#v", asset)
		}
	}
}

func TestAppDistCoreAsyncAndIODoNotPackageLibUVAssets(t *testing.T) {
	root := t.TempDir()
	assetDir := filepath.Join(root, "runtime", "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatalf("create runtime asset dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "os_cli_runtime.json"), []byte(`{"name":"os-cli"}`), 0o644); err != nil {
		t.Fatalf("write runtime asset: %v", err)
	}
	executable := filepath.Join(root, "build", "demo")
	writeFile(t, executable, "fake exe")

	plan := appdist.PlanApplicationWithStdlibImports(
		executable,
		filepath.Join(root, "dist", "demo"),
		binding.BuildPlan{},
		"linux-x64",
		[]string{
			"child_process",
			"fs",
			"microtask",
			"process",
			"stream",
			"tcp",
			"timer",
			"udp",
		},
		assetDir,
	)
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics for core async and I/O imports, got %#v", plan.Diagnostics)
	}
	for _, asset := range plan.Assets {
		if isNativeLibraryAsset(asset.OutputName) || isNativeLibraryAsset(asset.SourcePath) {
			t.Fatalf("core async and I/O import planned native library asset: %#v", asset)
		}
		if containsLibUVAsset(asset.OutputName) || containsLibUVAsset(filepath.Base(asset.SourcePath)) {
			t.Fatalf("core async and I/O import planned libuv asset: %#v", asset)
		}
	}
}

func TestAppDistCompressionDoesNotPackageExternalCompressionAssets(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	writeFile(t, executable, "fake exe")

	plan := appdist.PlanApplicationWithStdlibImports(
		executable,
		filepath.Join(root, "dist", "demo"),
		binding.BuildPlan{},
		"linux-x64",
		[]string{"compression"},
		filepath.Join(root, "missing"),
	)
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics for internal compression import, got %#v", plan.Diagnostics)
	}
	for _, asset := range plan.Assets {
		if isNativeLibraryAsset(asset.OutputName) || isNativeLibraryAsset(asset.SourcePath) {
			t.Fatalf("internal compression import planned native library asset: %#v", asset)
		}
		if containsExternalCompressionAsset(asset.OutputName) || containsExternalCompressionAsset(filepath.Base(asset.SourcePath)) {
			t.Fatalf("internal compression import planned external compression asset: %#v", asset)
		}
	}
}

func TestAppDistBrotliUnsupportedPolicyDoesNotPackageBrotliAssets(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	writeFile(t, executable, "fake exe")

	plan := appdist.PlanApplicationWithStdlibImports(
		executable,
		filepath.Join(root, "dist", "demo"),
		binding.BuildPlan{},
		"linux-x64",
		[]string{"compression"},
		filepath.Join(root, "missing"),
	)
	for _, asset := range plan.Assets {
		if containsExternalCompressionAsset(asset.OutputName) || containsExternalCompressionAsset(filepath.Base(asset.SourcePath)) {
			t.Fatalf("unsupported Brotli policy planned external compression asset: %#v", asset)
		}
	}
}

func TestAppDistStorageDoesNotPackageSQLiteAssets(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	writeFile(t, executable, "fake exe")

	plan := appdist.PlanApplicationWithStdlibImports(
		executable,
		filepath.Join(root, "dist", "demo"),
		binding.BuildPlan{},
		"linux-x64",
		[]string{"storage"},
		filepath.Join(root, "missing"),
	)
	if len(plan.Diagnostics) != 0 || len(plan.Assets) != 0 {
		t.Fatalf("expected internal storage to need no SQLite assets, got assets=%#v diagnostics=%#v", plan.Assets, plan.Diagnostics)
	}
}

func isNativeLibraryAsset(path string) bool {
	path = strings.ToLower(path)
	return strings.HasSuffix(path, ".so") ||
		strings.Contains(path, ".so.") ||
		strings.HasSuffix(path, ".dll") ||
		strings.HasSuffix(path, ".dylib")
}

func containsExternalNetworkingAsset(path string) bool {
	path = strings.ToLower(path)
	return strings.Contains(path, "curl") ||
		strings.Contains(path, "mongoose") ||
		strings.Contains(path, "picohttpparser")
}

func containsLibUVAsset(path string) bool {
	return strings.Contains(strings.ToLower(path), "libuv")
}

func containsExternalCompressionAsset(path string) bool {
	path = strings.ToLower(path)
	return strings.Contains(path, "zlib") ||
		strings.Contains(path, "brotli")
}
