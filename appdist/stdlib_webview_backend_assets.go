package appdist

import (
	"os"
	"path/filepath"

	runtimewebview "jayess-go/runtime/webview"
)

func resolveWebviewBackendRuntimeAssets(imports []string, targetName string, runtimeAssetDir string) ([]RuntimeAsset, []string) {
	if !usesWebviewStdlib(imports) {
		return nil, nil
	}
	requirement, ok := runtimewebview.BackendRequirementForTarget(targetName)
	if !ok || len(requirement.RedistributableAssets) == 0 {
		return nil, nil
	}
	var assets []RuntimeAsset
	var diagnostics []string
	for _, relative := range requirement.RedistributableAssets {
		source := filepath.Join(runtimeAssetDir, filepath.FromSlash(relative))
		if _, err := os.Stat(source); err != nil {
			diagnostics = append(diagnostics, "missing webview backend runtime asset: "+source)
			continue
		}
		assets = append(assets, RuntimeAsset{
			SourcePath: source,
			OutputName: filepath.Clean(filepath.FromSlash(relative)),
		})
	}
	return assets, diagnostics
}
