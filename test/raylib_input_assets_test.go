package test

import (
	"testing"

	"jayess-go/raylib"
)

func TestRaylibInputAndTimingFeatures(t *testing.T) {
	features := raylib.InputFeatures()
	for _, want := range []raylib.InputFeature{
		raylib.KeyboardInput,
		raylib.MouseInput,
		raylib.GamepadInput,
		raylib.FrameDeltaTiming,
		raylib.TargetFPSTiming,
		raylib.WindowModeSwitch,
	} {
		if !hasRaylibInputFeature(features, want) {
			t.Fatalf("expected raylib input feature %s in %#v", want, features)
		}
	}
}

func TestRaylibAssetAndMediaFeatures(t *testing.T) {
	features := raylib.AssetFeatures()
	for _, want := range []raylib.AssetFeature{
		raylib.LoadImageAssets,
		raylib.LoadTextureAssets,
		raylib.UnloadImageAssets,
		raylib.UnloadTextureAssets,
		raylib.AudioPlayback,
		raylib.GameAssetPathMapping,
	} {
		if !hasRaylibAssetFeature(features, want) {
			t.Fatalf("expected raylib asset feature %s in %#v", want, features)
		}
	}
}

func hasRaylibInputFeature(features []raylib.InputFeature, want raylib.InputFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasRaylibAssetFeature(features []raylib.AssetFeature, want raylib.AssetFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
