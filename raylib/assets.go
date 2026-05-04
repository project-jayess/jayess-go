package raylib

type AssetFeature string

const (
	LoadImageAssets      AssetFeature = "load-image-assets"
	LoadTextureAssets    AssetFeature = "load-texture-assets"
	UnloadImageAssets    AssetFeature = "unload-image-assets"
	UnloadTextureAssets  AssetFeature = "unload-texture-assets"
	AudioPlayback        AssetFeature = "audio-playback"
	GameAssetPathMapping AssetFeature = "game-asset-path-mapping"
)

func AssetFeatures() []AssetFeature {
	return []AssetFeature{
		LoadImageAssets,
		LoadTextureAssets,
		UnloadImageAssets,
		UnloadTextureAssets,
		AudioPlayback,
		GameAssetPathMapping,
	}
}
