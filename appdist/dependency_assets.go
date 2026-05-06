package appdist

import (
	"fmt"
	"os"
	"path/filepath"
)

func ResolveDependencyAssets(graph DependencyGraph, targetName string) ([]RuntimeAsset, []string) {
	var assets []RuntimeAsset
	var diagnostics []string
	seen := map[string]struct{}{}
	for _, dependency := range graph.Dependencies {
		switch dependency.Kind {
		case NativeBindingDependency:
			bindingAssets, bindingDiagnostics := ResolveRuntimeAssets(dependency.BindingPlan, targetName)
			diagnostics = append(diagnostics, bindingDiagnostics...)
			for _, asset := range bindingAssets {
				assets = appendAsset(assets, seen, asset)
			}
		case JayessPackageDependency, ExternalPackageDependency:
			packageAssets, packageDiagnostics := resolvePackageMetadataAssets(dependency)
			diagnostics = append(diagnostics, packageDiagnostics...)
			for _, asset := range packageAssets {
				assets = appendAsset(assets, seen, asset)
			}
		case SourceModuleDependency, BuiltinPackageDependency:
			continue
		default:
			diagnostics = append(diagnostics, fmt.Sprintf("unsupported imported dependency kind %q for %s", dependency.Kind, dependency.ImportPath))
		}
	}
	return assets, diagnostics
}

func resolvePackageMetadataAssets(dependency ImportedDependency) ([]RuntimeAsset, []string) {
	var assets []RuntimeAsset
	var diagnostics []string
	seen := map[string]struct{}{}
	root := dependency.PackageRoot
	if root == "" {
		root = filepath.Dir(dependency.ResolvedPath)
	}
	if root == "." || root == "" {
		diagnostics = append(diagnostics, "imported package "+dependency.ImportPath+" has no package root for distribution metadata")
		return nil, diagnostics
	}
	metadata := dependency.Metadata
	if metadata.empty() {
		loaded, err := LoadPackageMetadata(root)
		if err != nil {
			diagnostics = append(diagnostics, "missing package distribution metadata for "+dependency.ImportPath+": "+err.Error())
			return nil, diagnostics
		}
		metadata = loaded
	}
	licenseAssets, licenseDiagnostics := resolvePackageLicenseAssets(root, metadata.LicenseFiles)
	diagnostics = append(diagnostics, licenseDiagnostics...)
	for _, asset := range licenseAssets {
		assets = appendAsset(assets, seen, asset)
	}
	for _, asset := range append([]PackageAsset{}, append(metadata.RuntimeAssets, metadata.HelperAssets...)...) {
		if asset.BuildOnly {
			continue
		}
		if asset.RequiresLicense && len(metadata.LicenseFiles) == 0 {
			diagnostics = append(diagnostics, "imported dependency "+dependency.ImportPath+" asset "+asset.Path+" requires licenseFiles metadata")
		}
		resolved, assetDiagnostics := resolvePackageAsset(root, asset)
		diagnostics = append(diagnostics, assetDiagnostics...)
		if resolved.SourcePath != "" {
			assets = appendAsset(assets, seen, resolved)
		}
	}
	for _, system := range metadata.SystemDependencies {
		if !system.BuildOnly {
			diagnostics = append(diagnostics, "system dependency "+system.Name+" for "+dependency.ImportPath+" must be marked buildOnly or represented as a runtime asset")
		}
	}
	return assets, diagnostics
}

func (metadata PackageMetadata) empty() bool {
	return len(metadata.RuntimeAssets) == 0 &&
		len(metadata.HelperAssets) == 0 &&
		len(metadata.LicenseFiles) == 0 &&
		len(metadata.SystemDependencies) == 0
}

func resolvePackageLicenseAssets(root string, paths []string) ([]RuntimeAsset, []string) {
	var assets []RuntimeAsset
	var diagnostics []string
	for _, path := range paths {
		clean := filepath.Clean(filepath.Join(root, filepath.FromSlash(path)))
		if _, err := os.Stat(clean); err != nil {
			diagnostics = append(diagnostics, "missing imported dependency license file: "+clean)
			continue
		}
		assets = append(assets, RuntimeAsset{SourcePath: clean, OutputName: filepath.Join("licenses", filepath.Base(clean))})
	}
	return assets, diagnostics
}

func resolvePackageAsset(root string, asset PackageAsset) (RuntimeAsset, []string) {
	if asset.Path == "" {
		return RuntimeAsset{}, []string{"package runtime asset path must not be empty"}
	}
	source := filepath.Clean(filepath.Join(root, filepath.FromSlash(asset.Path)))
	if _, err := os.Stat(source); err != nil {
		return RuntimeAsset{}, []string{"missing imported runtime asset: " + source}
	}
	output := filepath.Clean(filepath.FromSlash(asset.OutputName))
	if output == "." || output == "" {
		output = filepath.Base(source)
	}
	if filepath.IsAbs(output) || output == ".." || len(output) >= 3 && output[:3] == "../" {
		return RuntimeAsset{}, []string{"package runtime asset outputName must be relative: " + asset.OutputName}
	}
	return RuntimeAsset{SourcePath: source, OutputName: output}, nil
}

func appendAsset(assets []RuntimeAsset, seen map[string]struct{}, asset RuntimeAsset) []RuntimeAsset {
	key := filepath.Clean(asset.SourcePath)
	if _, ok := seen[key]; ok {
		return assets
	}
	seen[key] = struct{}{}
	return append(assets, RuntimeAsset{SourcePath: key, OutputName: filepath.Clean(asset.OutputName)})
}
