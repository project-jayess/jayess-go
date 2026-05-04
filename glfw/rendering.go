package glfw

type RenderingFeature string

const (
	OpenGLFunctionAccess RenderingFeature = "opengl-function-access"
	VulkanIntegration    RenderingFeature = "vulkan-integration"
	TimingFrameLoop      RenderingFeature = "timing-frame-loop"
	ResizeHandling       RenderingFeature = "resize-handling"
	FullscreenSwitching  RenderingFeature = "fullscreen-switching"
)

func RenderingFeatures() []RenderingFeature {
	return []RenderingFeature{
		OpenGLFunctionAccess,
		VulkanIntegration,
		TimingFrameLoop,
		ResizeHandling,
		FullscreenSwitching,
	}
}
