package test

import (
	"testing"

	"jayess-go/raylib"
)

func TestRaylibHandlesAndColorValuesAreRepresentable(t *testing.T) {
	for _, kind := range []raylib.HandleKind{
		raylib.WindowHandle,
		raylib.ImageHandle,
		raylib.TextureHandle,
		raylib.SoundHandle,
		raylib.MusicHandle,
	} {
		if !raylib.SupportsHandle(kind) {
			t.Fatalf("expected raylib handle support for %s", kind)
		}
	}

	color := raylib.RGBA(10, 20, 30, 255)
	if color.R != 10 || color.G != 20 || color.B != 30 || color.A != 255 {
		t.Fatalf("unexpected color value: %#v", color)
	}
}

func TestRaylibSafetyFeaturesAndDiagnostics(t *testing.T) {
	safety := raylib.SafetyFeatures()
	for _, want := range []raylib.SafetyFeature{
		raylib.CallbackLifetimeSafety,
		raylib.ResourceHandleLifetime,
		raylib.AsyncRuntimeCoexist,
		raylib.DiagnosticPropagation,
	} {
		if !hasRaylibSafetyFeature(safety, want) {
			t.Fatalf("expected raylib safety feature %s in %#v", want, safety)
		}
	}

	diagnostics := raylib.DiagnosticKinds()
	for _, want := range []raylib.DiagnosticKind{
		raylib.MissingHeaders,
		raylib.MissingSource,
		raylib.MissingLibrary,
		raylib.BuildFailure,
	} {
		if !hasRaylibDiagnosticKind(diagnostics, want) {
			t.Fatalf("expected raylib diagnostic %s in %#v", want, diagnostics)
		}
	}
}

func hasRaylibSafetyFeature(features []raylib.SafetyFeature, want raylib.SafetyFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasRaylibDiagnosticKind(values []raylib.DiagnosticKind, want raylib.DiagnosticKind) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
