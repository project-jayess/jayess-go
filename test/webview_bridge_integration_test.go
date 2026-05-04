package test

import (
	"testing"

	"jayess-go/webview"
)

func TestWebviewBridgeFeatures(t *testing.T) {
	features := webview.BridgeFeatures()
	for _, want := range []webview.BridgeFeature{
		webview.ExposeJayessFunction,
		webview.ReceiveJavaScriptEvent,
		webview.SafeStringJSONBoundary,
		webview.SafeCallbackLifetime,
		webview.BridgeErrorPropagation,
	} {
		if !hasWebviewBridgeFeature(features, want) {
			t.Fatalf("expected webview bridge feature %s in %#v", want, features)
		}
	}
}

func TestWebviewAppIntegrationFeatures(t *testing.T) {
	features := webview.IntegrationFeatures()
	for _, want := range []webview.IntegrationFeature{
		webview.NativeHTTPServerIntegration,
		webview.WorkerThreadIntegration,
		webview.FilesystemPathIntegration,
		webview.GLFWHostIntegration,
		webview.GTKHostIntegration,
	} {
		if !hasWebviewIntegrationFeature(features, want) {
			t.Fatalf("expected webview integration feature %s in %#v", want, features)
		}
	}
}

func hasWebviewBridgeFeature(features []webview.BridgeFeature, want webview.BridgeFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasWebviewIntegrationFeature(features []webview.IntegrationFeature, want webview.IntegrationFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
