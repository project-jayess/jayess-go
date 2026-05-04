package test

import (
	"testing"

	"jayess-go/webview"
)

func TestWebviewLifecycleFeatures(t *testing.T) {
	features := webview.LifecycleFeatures()
	for _, want := range []webview.LifecycleFeature{
		webview.CreateWindow,
		webview.DestroyWindow,
		webview.SetWindowTitle,
		webview.SetWindowSize,
		webview.ShowWindow,
		webview.HideWindow,
		webview.EnterEventLoop,
		webview.CleanShutdown,
	} {
		if !hasWebviewLifecycleFeature(features, want) {
			t.Fatalf("expected webview lifecycle feature %s in %#v", want, features)
		}
	}
}

func TestWebviewContentFeatures(t *testing.T) {
	features := webview.ContentFeatures()
	for _, want := range []webview.ContentFeature{
		webview.LoadInlineHTML,
		webview.LoadLocalFile,
		webview.NavigateToURL,
		webview.ServeLocalHTTPApp,
		webview.InjectJavaScript,
	} {
		if !hasWebviewContentFeature(features, want) {
			t.Fatalf("expected webview content feature %s in %#v", want, features)
		}
	}
}

func hasWebviewLifecycleFeature(features []webview.LifecycleFeature, want webview.LifecycleFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasWebviewContentFeature(features []webview.ContentFeature, want webview.ContentFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
