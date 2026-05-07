package test

import (
	"strings"
	"testing"

	"jayess-go/webview"
)

func TestWebviewBridgeContractCoversWindowMountEventFlow(t *testing.T) {
	contract := webview.DefaultBridgeContract()
	for _, want := range []webview.HostCallKind{
		webview.CreateWindowCall,
		webview.MountContentCall,
		webview.DispatchEventCall,
		webview.EmitHostMessageCall,
	} {
		if !hasWebviewHostCall(contract.Calls, want) {
			t.Fatalf("expected bridge call %s in %#v", want, contract.Calls)
		}
	}
	for _, want := range []webview.EventKind{
		webview.WindowOpenedEvent,
		webview.WindowClosedEvent,
		webview.HostMessageEvent,
		webview.DialogResultEvent,
		webview.FileDropEvent,
	} {
		if !hasWebviewEvent(contract.Events, want) {
			t.Fatalf("expected bridge event %s in %#v", want, contract.Events)
		}
	}
}

func TestWebviewDialogAndDropFeaturesStayFocused(t *testing.T) {
	if !hasDialogFeature(webview.DialogFeatures(), webview.OpenFileDialog) {
		t.Fatalf("expected dialog feature %s", webview.OpenFileDialog)
	}
	if !hasDialogFeature(webview.DialogFeatures(), webview.SaveFileDialog) {
		t.Fatalf("expected dialog feature %s", webview.SaveFileDialog)
	}
	if !hasDropFeature(webview.DropFeatures(), webview.DropSourceFiles) {
		t.Fatalf("expected drop feature %s", webview.DropSourceFiles)
	}
	if !hasDropFeature(webview.DropFeatures(), webview.DropAppAssets) {
		t.Fatalf("expected drop feature %s", webview.DropAppAssets)
	}
}

func TestWebviewDiagnosticsExplainUnsupportedCases(t *testing.T) {
	if !strings.Contains(webview.UnsupportedPlatformDiagnostic("plan9"), "plan9") {
		t.Fatalf("unexpected platform diagnostic %q", webview.UnsupportedPlatformDiagnostic("plan9"))
	}
	if !strings.Contains(webview.UnsupportedCapabilityDiagnostic("save-file-dialog"), "save-file-dialog") {
		t.Fatalf("unexpected capability diagnostic %q", webview.UnsupportedCapabilityDiagnostic("save-file-dialog"))
	}
}

func hasWebviewHostCall(calls []webview.HostCallKind, want webview.HostCallKind) bool {
	for _, call := range calls {
		if call == want {
			return true
		}
	}
	return false
}

func hasWebviewEvent(events []webview.EventKind, want webview.EventKind) bool {
	for _, event := range events {
		if event == want {
			return true
		}
	}
	return false
}

func hasDialogFeature(features []webview.DialogFeature, want webview.DialogFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasDropFeature(features []webview.DropFeature, want webview.DropFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
