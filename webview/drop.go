package webview

type DropFeature string

const (
	DropSourceFiles DropFeature = "drop-source-files"
	DropAppAssets   DropFeature = "drop-app-assets"
)

func DropFeatures() []DropFeature {
	return []DropFeature{DropSourceFiles, DropAppAssets}
}
