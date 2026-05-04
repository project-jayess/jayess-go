package glfw

type IntegrationFeature string

const (
	ImageLoadingIntegration IntegrationFeature = "image-loading-integration"
	AudioLoopCoexistence    IntegrationFeature = "audio-loop-coexistence"
	WorkerRenderLoopInterop IntegrationFeature = "worker-render-loop-interop"
)

func IntegrationFeatures() []IntegrationFeature {
	return []IntegrationFeature{
		ImageLoadingIntegration,
		AudioLoopCoexistence,
		WorkerRenderLoopInterop,
	}
}
