package glfw

type WindowFeature string

const (
	InitializeGLFW      WindowFeature = "initialize-glfw"
	CreateWindow        WindowFeature = "create-window"
	DestroyWindow       WindowFeature = "destroy-window"
	PollEvents          WindowFeature = "poll-events"
	SwapBuffers         WindowFeature = "swap-buffers"
	CreateOpenGLContext WindowFeature = "create-opengl-context"
	CreateVulkanSurface WindowFeature = "create-vulkan-surface"
)

func WindowFeatures() []WindowFeature {
	return []WindowFeature{
		InitializeGLFW,
		CreateWindow,
		DestroyWindow,
		PollEvents,
		SwapBuffers,
		CreateOpenGLContext,
		CreateVulkanSurface,
	}
}
