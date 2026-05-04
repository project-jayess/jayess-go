package test

import (
	"testing"

	"jayess-go/raylib"
)

func TestRaylibWindowLifecycleFeatures(t *testing.T) {
	features := raylib.WindowFeatures()
	for _, want := range []raylib.WindowFeature{
		raylib.InitializeRaylib,
		raylib.CreateWindow,
		raylib.SetWindowTitle,
		raylib.SetWindowSize,
		raylib.WindowShouldClose,
		raylib.CloseWindow,
		raylib.FrameUpdateLoop,
	} {
		if !hasRaylibWindowFeature(features, want) {
			t.Fatalf("expected raylib window feature %s in %#v", want, features)
		}
	}
}

func TestRaylibRenderingFeatures(t *testing.T) {
	features := raylib.RenderingFeatures()
	for _, want := range []raylib.RenderingFeature{
		raylib.BeginDrawing,
		raylib.EndDrawing,
		raylib.ClearBackground,
		raylib.DrawText,
		raylib.DrawShapes,
		raylib.DrawTextures,
		raylib.PassColorValues,
	} {
		if !hasRaylibRenderingFeature(features, want) {
			t.Fatalf("expected raylib rendering feature %s in %#v", want, features)
		}
	}
}

func hasRaylibWindowFeature(features []raylib.WindowFeature, want raylib.WindowFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasRaylibRenderingFeature(features []raylib.RenderingFeature, want raylib.RenderingFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
