package test

import "testing"

func TestSemanticAllowsImportMetaExpression(t *testing.T) {
	err := analyzeSource(t, `
		function main() {
			import.meta.url;
			return 0;
		}
	`)
	if err != nil {
		t.Fatalf("expected import.meta to analyze, got %v", err)
	}
}
