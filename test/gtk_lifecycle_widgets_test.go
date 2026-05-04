package test

import (
	"testing"

	"jayess-go/gtk"
)

func TestGTKLifecycleFeatures(t *testing.T) {
	features := gtk.LifecycleFeatures()
	for _, want := range []gtk.LifecycleFeature{
		gtk.InitializeRuntime,
		gtk.CreateApplication,
		gtk.CreateWindow,
		gtk.EnterMainLoop,
		gtk.QuitMainLoop,
		gtk.CleanShutdown,
	} {
		if !hasGTKLifecycleFeature(features, want) {
			t.Fatalf("expected GTK lifecycle feature %s in %#v", want, features)
		}
	}
}

func TestGTKWidgetFeatures(t *testing.T) {
	features := gtk.WidgetFeatures()
	for _, want := range []gtk.WidgetFeature{
		gtk.CreateLabel,
		gtk.CreateButton,
		gtk.CreateTextInput,
		gtk.CreateContainer,
		gtk.SetProperty,
		gtk.AddChild,
		gtk.ShowWidget,
		gtk.HideWidget,
	} {
		if !hasGTKWidgetFeature(features, want) {
			t.Fatalf("expected GTK widget feature %s in %#v", want, features)
		}
	}
}

func hasGTKLifecycleFeature(features []gtk.LifecycleFeature, want gtk.LifecycleFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasGTKWidgetFeature(features []gtk.WidgetFeature, want gtk.WidgetFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
