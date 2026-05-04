package test

import (
	"testing"

	"jayess-go/glfw"
)

func TestGLFWInputFeatures(t *testing.T) {
	features := glfw.InputFeatures()
	for _, want := range []glfw.InputFeature{
		glfw.KeyboardCallback,
		glfw.MouseButtonCallback,
		glfw.CursorPositionCallback,
		glfw.ScrollCallback,
		glfw.GamepadJoystickInput,
	} {
		if !hasGLFWInputFeature(features, want) {
			t.Fatalf("expected GLFW input feature %s in %#v", want, features)
		}
	}
}

func TestGLFWMediaIntegrationFeatures(t *testing.T) {
	features := glfw.IntegrationFeatures()
	for _, want := range []glfw.IntegrationFeature{
		glfw.ImageLoadingIntegration,
		glfw.AudioLoopCoexistence,
		glfw.WorkerRenderLoopInterop,
	} {
		if !hasGLFWIntegrationFeature(features, want) {
			t.Fatalf("expected GLFW integration feature %s in %#v", want, features)
		}
	}
}

func hasGLFWInputFeature(features []glfw.InputFeature, want glfw.InputFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasGLFWIntegrationFeature(features []glfw.IntegrationFeature, want glfw.IntegrationFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
