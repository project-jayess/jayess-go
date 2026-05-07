package test

import (
	"path/filepath"
	"testing"

	runtimewebview "jayess-go/runtime/webview"
	"jayess-go/webview"
)

func TestWebviewDefaultContentMountsCoverEmbeddedAndGeneratedContent(t *testing.T) {
	mounts := webview.DefaultContentMounts()
	if len(mounts) != 3 {
		t.Fatalf("expected three default content mounts, got %#v", mounts)
	}
	requireWebviewMount(t, mounts, webview.EmbeddedDocumentMount)
	requireWebviewMount(t, mounts, webview.EmbeddedAssetsMount)
	requireWebviewMount(t, mounts, webview.GeneratedContentMount)
}

func TestWebviewRuntimeAssetPathsStayExplicit(t *testing.T) {
	if webview.RuntimeAssetOutputName != "runtime/webview_runtime.json" {
		t.Fatalf("unexpected webview runtime asset output %q", webview.RuntimeAssetOutputName)
	}
	source := webview.RuntimeAssetSourcePath(filepath.Join("runtime", "assets"))
	if source != filepath.Join("runtime", "assets", "webview_runtime.json") {
		t.Fatalf("unexpected webview runtime asset source %q", source)
	}
	if runtimewebview.DefaultSupport().RuntimeAssetOutput != filepath.Join("runtime", "webview_runtime.json") {
		t.Fatalf("unexpected runtime support output path %q", runtimewebview.DefaultSupport().RuntimeAssetOutput)
	}
	if runtimewebview.DefaultSupport().RequiresPackageInstall {
		t.Fatal("webview runtime support should not require separate package installation")
	}
}

func requireWebviewMount(t *testing.T, mounts []webview.ContentMount, want webview.MountKind) {
	t.Helper()
	for _, mount := range mounts {
		if mount.Kind == want {
			return
		}
	}
	t.Fatalf("expected mount %s in %#v", want, mounts)
}
