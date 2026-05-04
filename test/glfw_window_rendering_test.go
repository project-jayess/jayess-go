package test

import (
	"testing"

	"jayess-go/glfw"
)

func TestGLFWWindowAndContextFeatures(t *testing.T) {
	features := glfw.WindowFeatures()
	for _, want := range []glfw.WindowFeature{
		glfw.InitializeGLFW,
		glfw.CreateWindow,
		glfw.DestroyWindow,
		glfw.PollEvents,
		glfw.SwapBuffers,
		glfw.CreateOpenGLContext,
		glfw.CreateVulkanSurface,
	} {
		if !hasGLFWWindowFeature(features, want) {
			t.Fatalf("expected GLFW window feature %s in %#v", want, features)
		}
	}
}

func TestGLFWRenderingFeatures(t *testing.T) {
	features := glfw.RenderingFeatures()
	for _, want := range []glfw.RenderingFeature{
		glfw.OpenGLFunctionAccess,
		glfw.VulkanIntegration,
		glfw.TimingFrameLoop,
		glfw.ResizeHandling,
		glfw.FullscreenSwitching,
	} {
		if !hasGLFWRenderingFeature(features, want) {
			t.Fatalf("expected GLFW rendering feature %s in %#v", want, features)
		}
	}
}

func hasGLFWWindowFeature(features []glfw.WindowFeature, want glfw.WindowFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasGLFWRenderingFeature(features []glfw.RenderingFeature, want glfw.RenderingFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
