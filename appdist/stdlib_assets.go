package appdist

import (
	"os"
	"path/filepath"
	"sort"

	"jayess-go/binding"
)

var osCLIStdlibImports = map[string]struct{}{
	"child_process": {},
	"fs":            {},
	"process":       {},
	"stream":        {},
	"terminal":      {},
}

func ResolveStdlibRuntimeAssets(imports []string, targetName string, runtimeAssetDir string) ([]RuntimeAsset, []string) {
	if runtimeAssetDir == "" {
		runtimeAssetDir = filepath.Join("runtime", "assets")
	}
	var assets []RuntimeAsset
	var diagnostics []string
	if usesOSCLIStdlib(imports) {
		source := filepath.Join(runtimeAssetDir, "os_cli_runtime.json")
		if _, err := os.Stat(source); err != nil {
			diagnostics = append(diagnostics, "missing OS/CLI runtime asset: "+source)
		} else {
			assets = append(assets, RuntimeAsset{
				SourcePath: source,
				OutputName: filepath.Join("runtime", "os_cli_runtime.json"),
			})
		}
	}
	webviewAssets, webviewDiagnostics := resolveWebviewRuntimeAssets(imports, runtimeAssetDir)
	assets = appendRuntimeAssets(assets, webviewAssets...)
	diagnostics = append(diagnostics, webviewDiagnostics...)
	webviewBackendAssets, webviewBackendDiagnostics := resolveWebviewBackendRuntimeAssets(imports, targetName, runtimeAssetDir)
	assets = appendRuntimeAssets(assets, webviewBackendAssets...)
	diagnostics = append(diagnostics, webviewBackendDiagnostics...)
	if len(assets) == 0 && len(diagnostics) == 0 {
		return nil, nil
	}
	return assets, diagnostics
}

func PlanApplicationWithStdlibImports(executablePath string, outputDir string, bindingPlan binding.BuildPlan, targetName string, imports []string, runtimeAssetDir string) Plan {
	plan := PlanApplication(executablePath, outputDir, bindingPlan, targetName)
	stdlibAssets, diagnostics := ResolveStdlibRuntimeAssets(imports, targetName, runtimeAssetDir)
	plan.Assets = appendRuntimeAssets(plan.Assets, stdlibAssets...)
	plan.Diagnostics = append(plan.Diagnostics, diagnostics...)
	return plan
}

func usesOSCLIStdlib(imports []string) bool {
	for _, importPath := range imports {
		if _, ok := osCLIStdlibImports[importPath]; ok {
			return true
		}
	}
	return false
}

func appendRuntimeAssets(assets []RuntimeAsset, extra ...RuntimeAsset) []RuntimeAsset {
	seen := map[string]struct{}{}
	for _, asset := range assets {
		seen[asset.SourcePath+"=>"+asset.OutputName] = struct{}{}
	}
	for _, asset := range extra {
		key := asset.SourcePath + "=>" + asset.OutputName
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		assets = append(assets, asset)
	}
	sort.SliceStable(assets, func(i, j int) bool {
		return assets[i].OutputName < assets[j].OutputName
	})
	return assets
}
