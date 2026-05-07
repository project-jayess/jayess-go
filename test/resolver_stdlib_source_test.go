package test

import (
	"path/filepath"
	"testing"

	"jayess-go/resolver"
)

func TestResolverMapsWebviewStdlibImportToVendoredSource(t *testing.T) {
	sourcePath, ok, err := resolver.ResolvedStdlibSourcePath("jayess:stdlib/@jayess/webview")
	if err != nil {
		t.Fatalf("ResolvedStdlibSourcePath returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected source-backed stdlib mapping")
	}
	requireResolvedSuffix(t, sourcePath, filepath.Join("stdlib", "@jayess", "webview", "index.js"))
}
