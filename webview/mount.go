package webview

type MountKind string

const (
	EmbeddedDocumentMount MountKind = "embedded-document"
	EmbeddedAssetsMount   MountKind = "embedded-assets"
	GeneratedContentMount MountKind = "generated-content"
)

type AssetKind string

const (
	HTMLAsset   AssetKind = "html"
	CSSAsset    AssetKind = "css"
	ScriptAsset AssetKind = "script"
	StaticAsset AssetKind = "static"
)

type ContentMount struct {
	Kind       MountKind
	AssetKinds []AssetKind
}

func DefaultContentMounts() []ContentMount {
	return []ContentMount{
		{Kind: EmbeddedDocumentMount, AssetKinds: []AssetKind{HTMLAsset}},
		{Kind: EmbeddedAssetsMount, AssetKinds: []AssetKind{HTMLAsset, CSSAsset, ScriptAsset, StaticAsset}},
		{Kind: GeneratedContentMount, AssetKinds: []AssetKind{HTMLAsset, CSSAsset, ScriptAsset}},
	}
}
