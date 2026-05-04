package test

import (
	"testing"

	"jayess-go/webparse"
)

func TestWebParseIntegrationFeaturesAndFileParsing(t *testing.T) {
	doc, err := webparse.ParseHTMLFromFile("index.html", func(path string) (string, error) {
		if path != "index.html" {
			t.Fatalf("unexpected path %s", path)
		}
		return `<h1>Hello</h1>`, nil
	})
	if err != nil {
		t.Fatalf("expected file parse, got %v", err)
	}
	if doc.Root.Children[0].Span.File != "index.html" {
		t.Fatalf("expected file span, got %#v", doc.Root.Children[0].Span)
	}

	features := webparse.IntegrationFeatures()
	for _, want := range []webparse.IntegrationFeature{
		webparse.FileSystemParsing,
		webparse.ModuleSystemParsing,
		webparse.UserProgramASTNodes,
		webparse.CompilerDiagnostics,
		webparse.CompilerSpanAlignment,
	} {
		if !hasWebParseIntegrationFeature(features, want) {
			t.Fatalf("expected webparse integration feature %s in %#v", want, features)
		}
	}
}

func hasWebParseIntegrationFeature(values []webparse.IntegrationFeature, want webparse.IntegrationFeature) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
