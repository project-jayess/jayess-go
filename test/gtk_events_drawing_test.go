package test

import (
	"testing"

	"jayess-go/gtk"
)

func TestGTKEventFeatures(t *testing.T) {
	features := gtk.EventFeatures()
	for _, want := range []gtk.EventFeature{
		gtk.ConnectSignal,
		gtk.ButtonClick,
		gtk.InputChange,
		gtk.WindowClose,
		gtk.SafeCallback,
	} {
		if !hasGTKEventFeature(features, want) {
			t.Fatalf("expected GTK event feature %s in %#v", want, features)
		}
	}
}

func TestGTKDrawingFeatures(t *testing.T) {
	features := gtk.DrawingFeatures()
	for _, want := range []gtk.DrawingFeature{
		gtk.ImageAssetLoading,
		gtk.CustomDrawing,
		gtk.TextRendering,
	} {
		if !hasGTKDrawingFeature(features, want) {
			t.Fatalf("expected GTK drawing feature %s in %#v", want, features)
		}
	}
}

func hasGTKEventFeature(features []gtk.EventFeature, want gtk.EventFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasGTKDrawingFeature(features []gtk.DrawingFeature, want gtk.DrawingFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
