package webview

import "path/filepath"

const (
	PackageImport         = "@jayess/webview"
	RuntimeAssetFile      = "webview_runtime.json"
	RuntimeAssetDirectory = "runtime/assets"
)

func RuntimeAssetOutputPath() string {
	return filepath.Join("runtime", RuntimeAssetFile)
}
