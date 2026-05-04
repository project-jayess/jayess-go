package test

import "testing"

func TestSemanticDeclaresNamespaceImport(t *testing.T) {
	err := analyzeSource(t, `
		import * as math from "./math.js";
		const value = math.add(1, 2);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsDuplicateNamespaceImportLocal(t *testing.T) {
	err := analyzeSource(t, `
		const math = 1;
		import * as math from "./math.js";
	`)
	requireSemanticError(t, err, "duplicate declaration math")
}
