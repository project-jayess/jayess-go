package test

import (
	"os"
	"path/filepath"
	"testing"

	"jayess-go/appdist"
	"jayess-go/binding"
)

func TestAppDistCopiesExecutableAndSharedLibraries(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	sharedLibrary := filepath.Join(root, "sdk", "packages", "glfw", "lib", "libglfw.so")
	writeFile(t, executable, "fake exe")
	writeFile(t, sharedLibrary, "fake glfw")

	bindingPlan := binding.BuildPlan{
		LibraryDirs:     []string{filepath.Dir(sharedLibrary)},
		SharedLibraries: []string{"glfw"},
	}
	plan := appdist.PlanApplication(executable, filepath.Join(root, "dist", "demo"), bindingPlan, "linux-x64")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", plan.Diagnostics)
	}
	result, err := appdist.Create(plan)
	if err != nil {
		t.Fatal(err)
	}
	requireFile(t, filepath.Join(result.OutputDir, "demo"))
	requireFile(t, filepath.Join(result.OutputDir, "libglfw.so"))
	if len(result.CopiedAssetPaths) != 1 {
		t.Fatalf("expected one copied runtime asset, got %#v", result.CopiedAssetPaths)
	}
}

func TestAppDistIgnoresStaticLibrariesAndReportsMissingSharedLibraries(t *testing.T) {
	root := t.TempDir()
	staticLibrary := filepath.Join(root, "sdk", "packages", "glfw", "lib", "libglfw.a")
	writeFile(t, staticLibrary, "fake static glfw")

	bindingPlan := binding.BuildPlan{
		LibraryDirs:     []string{filepath.Dir(staticLibrary)},
		SharedLibraries: []string{"glfw", staticLibrary},
	}
	assets, diagnostics := appdist.ResolveRuntimeAssets(bindingPlan, "linux-x64")
	if len(assets) != 0 {
		t.Fatalf("expected no runtime assets for static-only library, got %#v", assets)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected missing shared library diagnostic, got %#v", diagnostics)
	}
}

func TestAppDistResolvesWindowsRuntimeDLL(t *testing.T) {
	root := t.TempDir()
	dll := filepath.Join(root, "sdk", "packages", "glfw", "lib", "glfw.dll")
	writeFile(t, dll, "fake dll")

	bindingPlan := binding.BuildPlan{
		LibraryDirs:     []string{filepath.Dir(dll)},
		SharedLibraries: []string{"glfw"},
	}
	assets, diagnostics := appdist.ResolveRuntimeAssets(bindingPlan, "windows-x64")
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if len(assets) != 1 || assets[0].OutputName != "glfw.dll" {
		t.Fatalf("unexpected windows assets: %#v", assets)
	}
}

func TestAppDistDoesNotRequireSeparateLibraryInstallForExplicitSharedLibrary(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	sharedLibrary := filepath.Join(root, "vendor", "libhelper.so")
	writeFile(t, executable, "fake exe")
	writeFile(t, sharedLibrary, "fake helper")

	bindingPlan := binding.BuildPlan{
		SharedLibraries: []string{sharedLibrary},
	}
	result, err := appdist.Create(appdist.PlanApplication(executable, filepath.Join(root, "dist", "demo"), bindingPlan, "linux-x64"))
	if err != nil {
		t.Fatal(err)
	}
	requireFile(t, filepath.Join(result.OutputDir, "demo"))
	requireFile(t, filepath.Join(result.OutputDir, "libhelper.so"))
}

func TestAppDistCollectsBindingOwnedSharedLibraryFiles(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	modulePath := filepath.Join(root, "src", "native", "helper.js")
	sharedLibrary := filepath.Join(root, "src", "native", "lib", "libhelper.so")
	licenseFile := filepath.Join(root, "src", "native", "LICENSE.helper")
	writeFile(t, executable, "fake exe")
	writeFile(t, sharedLibrary, "fake helper")
	writeFile(t, licenseFile, "helper license")

	bindingPlan := binding.PlanBuild([]binding.Module{{
		Path: modulePath,
		Manifest: binding.Manifest{
			Sources:         []string{"./helper.c"},
			SharedLibraries: []string{"./lib/libhelper.so"},
			LicenseFiles:    []string{"./LICENSE.helper"},
			Exports:         []binding.Export{{Name: "help", Symbol: "helper_help", Kind: binding.FunctionExport}},
		},
	}}, "linux", "")

	result, err := appdist.Create(appdist.PlanApplication(executable, filepath.Join(root, "dist", "demo"), bindingPlan, "linux-x64"))
	if err != nil {
		t.Fatal(err)
	}
	requireFile(t, filepath.Join(result.OutputDir, "demo"))
	requireFile(t, filepath.Join(result.OutputDir, "libhelper.so"))
	requireFile(t, filepath.Join(result.OutputDir, "licenses", "LICENSE.helper"))
	if len(result.CopiedAssetPaths) != 2 {
		t.Fatalf("expected shared library and license assets, got %#v", result.CopiedAssetPaths)
	}
}

func TestAppDistCreateRequiresExecutable(t *testing.T) {
	root := t.TempDir()
	_, err := appdist.Create(appdist.Plan{ExecutablePath: filepath.Join(root, "missing"), OutputDir: filepath.Join(root, "dist")})
	if err == nil {
		t.Fatal("expected missing executable error")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
}
