package test

import (
	"testing"

	runtimewebview "jayess-go/runtime/webview"
)

func TestRuntimeWebviewHostManagesWindowLifecycle(t *testing.T) {
	host := runtimewebview.NewHost()
	window := host.CreateWindow("Jayess", runtimewebview.Size{Width: 1024, Height: 768})

	if err := host.ShowWindow(window.ID); err != nil {
		t.Fatalf("ShowWindow returned error: %v", err)
	}
	if err := host.SetWindowTitle(window.ID, "Jayess GUI"); err != nil {
		t.Fatalf("SetWindowTitle returned error: %v", err)
	}
	if err := host.SetWindowSize(window.ID, runtimewebview.Size{Width: 1280, Height: 720}); err != nil {
		t.Fatalf("SetWindowSize returned error: %v", err)
	}
	if err := host.CloseWindow(window.ID); err != nil {
		t.Fatalf("CloseWindow returned error: %v", err)
	}

	current, ok := host.Window(window.ID)
	if !ok {
		t.Fatalf("expected window %s", window.ID)
	}
	if current.Title != "Jayess GUI" || current.State != runtimewebview.WindowClosed {
		t.Fatalf("unexpected window state %#v", current)
	}
}

func TestRuntimeWebviewHostMountsContentAndDeliversEvents(t *testing.T) {
	host := runtimewebview.NewHost()
	window := host.CreateWindow("Jayess", runtimewebview.Size{})
	mount := runtimewebview.Mount{
		Kind:   "embedded-assets",
		HTML:   "<main>jayess</main>",
		CSS:    "main { color: black; }",
		Script: "console.log('jayess');",
		Assets: []runtimewebview.Asset{{Path: "ui/logo.svg", Kind: "static"}},
	}

	if err := host.MountContent(window.ID, mount); err != nil {
		t.Fatalf("MountContent returned error: %v", err)
	}
	host.EmitHostMessage(window.ID, "ready")
	host.QueueFileDrop(window.ID, []string{"src/main.js", "assets/icon.svg"})

	current, ok := host.MountedContent(window.ID)
	if !ok {
		t.Fatal("expected mounted content")
	}
	if current.Kind != "embedded-assets" || len(current.Assets) != 1 {
		t.Fatalf("unexpected mount %#v", current)
	}

	events := host.DrainEvents()
	requireRuntimeWebviewEvent(t, events, "window-opened")
	requireRuntimeWebviewEvent(t, events, "navigation")
	requireRuntimeWebviewEvent(t, events, "host-message")
	requireRuntimeWebviewEvent(t, events, "file-drop")
}

func TestRuntimeWebviewHostQueuesAndCompletesDialogs(t *testing.T) {
	host := runtimewebview.NewHost()
	window := host.CreateWindow("Jayess", runtimewebview.Size{})

	if err := host.OpenFileDialog(window.ID, runtimewebview.DialogRequest{
		Title:       "Open source",
		DefaultPath: "src",
		Filters:     []string{".js", ".jayess"},
	}); err != nil {
		t.Fatalf("OpenFileDialog returned error: %v", err)
	}

	request, ok := host.PendingDialog(window.ID)
	if !ok || request.Kind != runtimewebview.OpenDialogKind {
		t.Fatalf("unexpected pending dialog %#v ok=%v", request, ok)
	}

	if err := host.CompleteDialog(window.ID, runtimewebview.DialogResult{
		Accepted: true,
		Path:     "src/main.js",
	}); err != nil {
		t.Fatalf("CompleteDialog returned error: %v", err)
	}

	events := host.DrainEvents()
	requireRuntimeWebviewEvent(t, events, "dialog-result")
}

func TestRuntimeWebviewSupportStaysInternalAndInstallFree(t *testing.T) {
	support := runtimewebview.DefaultSupport()
	if !support.UsesInternalRuntime {
		t.Fatal("expected internal runtime support")
	}
	if support.RequiresPackageInstall || support.RequiresEndUserInstall || support.ShipsThirdPartyGUILibs {
		t.Fatalf("unexpected support flags %#v", support)
	}
}

func requireRuntimeWebviewEvent(t *testing.T, events []runtimewebview.Event, want string) {
	t.Helper()
	for _, event := range events {
		if event.Kind == want {
			return
		}
	}
	t.Fatalf("expected event %q in %#v", want, events)
}
