package gtk

type DrawingFeature string

const (
	ImageAssetLoading DrawingFeature = "image-asset-loading"
	CustomDrawing     DrawingFeature = "custom-drawing"
	TextRendering     DrawingFeature = "text-rendering"
)

func DrawingFeatures() []DrawingFeature {
	return []DrawingFeature{
		ImageAssetLoading,
		CustomDrawing,
		TextRendering,
	}
}
