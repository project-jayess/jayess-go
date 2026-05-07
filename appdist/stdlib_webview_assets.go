package appdist

import (
	"os"
	"path/filepath"

	runtimewebview "jayess-go/runtime/webview"
)

func resolveWebviewRuntimeAssets(imports []string, runtimeAssetDir string) ([]RuntimeAsset, []string) {
	if !usesWebviewStdlib(imports) {
		return nil, nil
	}
	source := filepath.Join(runtimeAssetDir, runtimewebview.RuntimeAssetFile)
	if _, err := os.Stat(source); err != nil {
		return nil, []string{"missing webview runtime asset: " + source}
	}
	return []RuntimeAsset{{
		SourcePath: source,
		OutputName: runtimewebview.RuntimeAssetOutputPath(),
	}}, nil
}

func usesWebviewStdlib(imports []string) bool {
	for _, importPath := range imports {
		if importPath == runtimewebview.PackageImport {
			return true
		}
	}
	return false
}
