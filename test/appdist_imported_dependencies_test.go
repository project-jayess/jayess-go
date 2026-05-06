package test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/appdist"
	"jayess-go/binding"
)

func TestAppDistImportedNativeBindingShipsLibraryAssetAndLicense(t *testing.T) {
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
			SharedLibraries: []string{"./lib/libhelper.so"},
			LicenseFiles:    []string{"./LICENSE.helper"},
			Exports:         []binding.Export{{Name: "help", Symbol: "helper_help", Kind: binding.FunctionExport}},
		},
	}}, "linux-x64", "")

	plan := appdist.PlanApplicationFromDependencies(executable, filepath.Join(root, "dist", "demo"), appdist.DependencyGraph{
		Dependencies: []appdist.ImportedDependency{{
			ImportPath:   "./native/helper.js",
			ResolvedPath: modulePath,
			Kind:         appdist.NativeBindingDependency,
			BindingPlan:  bindingPlan,
		}},
	}, "linux-x64")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", plan.Diagnostics)
	}
	result, err := appdist.Create(plan)
	if err != nil {
		t.Fatal(err)
	}
	requireFile(t, filepath.Join(result.OutputDir, "demo"))
	requireFile(t, filepath.Join(result.OutputDir, "libhelper.so"))
	requireFile(t, filepath.Join(result.OutputDir, "licenses", "LICENSE.helper"))
}

func TestAppDistImportedPackageShipsDeclaredAssets(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	packageRoot := filepath.Join(root, "node_modules", "@jayess", "assets")
	writeFile(t, executable, "fake exe")
	writeFile(t, filepath.Join(packageRoot, "data", "schema.json"), "{}")
	writeFile(t, filepath.Join(packageRoot, "bin", "helper"), "helper")
	writeFile(t, filepath.Join(packageRoot, "LICENSE.assets"), "asset license")
	writeFile(t, filepath.Join(packageRoot, appdist.PackageMetadataFile), `{
  "runtimeAssets": [{"path": "data/schema.json", "outputName": "assets/schema.json", "requiresLicense": true}],
  "helperAssets": [{"path": "bin/helper", "outputName": "helpers/helper"}],
  "licenseFiles": ["LICENSE.assets"]
}`)

	plan := appdist.PlanApplicationFromDependencies(executable, filepath.Join(root, "dist", "demo"), appdist.DependencyGraph{
		Dependencies: []appdist.ImportedDependency{{
			ImportPath:  "@jayess/assets",
			PackageRoot: packageRoot,
			Kind:        appdist.JayessPackageDependency,
		}},
	}, "linux-x64")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", plan.Diagnostics)
	}
	result, err := appdist.Create(plan)
	if err != nil {
		t.Fatal(err)
	}
	requireFile(t, filepath.Join(result.OutputDir, "assets", "schema.json"))
	requireFile(t, filepath.Join(result.OutputDir, "helpers", "helper"))
	requireFile(t, filepath.Join(result.OutputDir, "licenses", "LICENSE.assets"))
}

func TestAppDistUnusedPackageDoesNotAddAssets(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	unusedRoot := filepath.Join(root, "node_modules", "@jayess", "unused")
	writeFile(t, executable, "fake exe")
	writeFile(t, filepath.Join(unusedRoot, "data", "unused.dat"), "unused")
	writeFile(t, filepath.Join(unusedRoot, appdist.PackageMetadataFile), `{
  "runtimeAssets": [{"path": "data/unused.dat"}]
}`)

	plan := appdist.PlanApplicationFromDependencies(executable, filepath.Join(root, "dist", "demo"), appdist.DependencyGraph{}, "linux-x64")
	if len(plan.Assets) != 0 {
		t.Fatalf("expected unused package asset to be excluded, got %#v", plan.Assets)
	}
}

func TestAppDistDeduplicatesImportedDependencyAssets(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	packageRoot := filepath.Join(root, "node_modules", "shared")
	writeFile(t, executable, "fake exe")
	writeFile(t, filepath.Join(packageRoot, "lib", "shared.dat"), "shared")
	metadata := appdist.PackageMetadata{
		RuntimeAssets: []appdist.PackageAsset{{Path: "lib/shared.dat", OutputName: "shared.dat"}},
	}

	plan := appdist.PlanApplicationFromDependencies(executable, filepath.Join(root, "dist", "demo"), appdist.DependencyGraph{
		Dependencies: []appdist.ImportedDependency{
			{ImportPath: "shared/a", PackageRoot: packageRoot, Kind: appdist.JayessPackageDependency, Metadata: metadata},
			{ImportPath: "shared/b", PackageRoot: packageRoot, Kind: appdist.JayessPackageDependency, Metadata: metadata},
		},
	}, "linux-x64")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", plan.Diagnostics)
	}
	if len(plan.Assets) != 1 {
		t.Fatalf("expected one deduplicated asset, got %#v", plan.Assets)
	}
}

func TestAppDistMissingImportedRuntimeAssetFailsPlanning(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	packageRoot := filepath.Join(root, "node_modules", "broken")
	writeFile(t, executable, "fake exe")

	plan := appdist.PlanApplicationFromDependencies(executable, filepath.Join(root, "dist", "demo"), appdist.DependencyGraph{
		Dependencies: []appdist.ImportedDependency{{
			ImportPath:  "broken",
			PackageRoot: packageRoot,
			Kind:        appdist.JayessPackageDependency,
			Metadata: appdist.PackageMetadata{
				RuntimeAssets: []appdist.PackageAsset{{Path: "missing.dat"}},
			},
		}},
	}, "linux-x64")
	if len(plan.Diagnostics) != 1 || !strings.Contains(plan.Diagnostics[0], "missing imported runtime asset") {
		t.Fatalf("expected missing asset diagnostic, got %#v", plan.Diagnostics)
	}
}

func TestAppDistBuildOnlySystemDependencyDoesNotRequireEndUserInstall(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	packageRoot := filepath.Join(root, "node_modules", "webview")
	writeFile(t, executable, "fake exe")

	plan := appdist.PlanApplicationFromDependencies(executable, filepath.Join(root, "dist", "demo"), appdist.DependencyGraph{
		Dependencies: []appdist.ImportedDependency{{
			ImportPath:  "webview",
			PackageRoot: packageRoot,
			Kind:        appdist.ExternalPackageDependency,
			Metadata: appdist.PackageMetadata{
				SystemDependencies: []appdist.SystemDependency{{Name: "Apple SDK", BuildOnly: true}},
			},
		}},
	}, "linux-x64")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics for build-only SDK input: %#v", plan.Diagnostics)
	}
}

func TestAppDistPackagedAppRunsFromDistributionDirectory(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo.sh")
	packageRoot := filepath.Join(root, "node_modules", "runner")
	writeFile(t, executable, "#!/bin/sh\ntest -f assets/config.json\n")
	writeFile(t, filepath.Join(packageRoot, "assets", "config.json"), "{}")

	plan := appdist.PlanApplicationFromDependencies(executable, filepath.Join(root, "dist", "demo"), appdist.DependencyGraph{
		Dependencies: []appdist.ImportedDependency{{
			ImportPath:  "runner",
			PackageRoot: packageRoot,
			Kind:        appdist.JayessPackageDependency,
			Metadata: appdist.PackageMetadata{
				RuntimeAssets: []appdist.PackageAsset{{Path: "assets/config.json", OutputName: "assets/config.json"}},
			},
		}},
	}, "linux-x64")
	result, err := appdist.Create(plan)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("./demo.sh")
	command.Dir = result.OutputDir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("expected packaged app to run from distribution directory: %v\n%s", err, string(output))
	}
}

func TestAppDistRuntimeAssetThatRequiresLicenseReportsMissingMetadata(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "build", "demo")
	packageRoot := filepath.Join(root, "node_modules", "licensed")
	writeFile(t, executable, "fake exe")
	writeFile(t, filepath.Join(packageRoot, "lib", "licensed.so"), "library")

	plan := appdist.PlanApplicationFromDependencies(executable, filepath.Join(root, "dist", "demo"), appdist.DependencyGraph{
		Dependencies: []appdist.ImportedDependency{{
			ImportPath:  "licensed",
			PackageRoot: packageRoot,
			Kind:        appdist.ExternalPackageDependency,
			Metadata: appdist.PackageMetadata{
				RuntimeAssets: []appdist.PackageAsset{{Path: "lib/licensed.so", RequiresLicense: true}},
			},
		}},
	}, "linux-x64")
	if len(plan.Diagnostics) != 1 || !strings.Contains(plan.Diagnostics[0], "requires licenseFiles metadata") {
		t.Fatalf("expected missing license metadata diagnostic, got %#v", plan.Diagnostics)
	}
}
