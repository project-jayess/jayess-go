package webview

import "path/filepath"

const (
	RuntimeAssetFile       = "webview_runtime.json"
	RuntimeAssetOutputName = "runtime/webview_runtime.json"
)

func RuntimeAssetSourcePath(runtimeAssetDir string) string {
	return filepath.Join(runtimeAssetDir, RuntimeAssetFile)
}
