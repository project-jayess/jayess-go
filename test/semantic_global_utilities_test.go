package test

import "testing"

func TestSemanticAllowsGlobalThis(t *testing.T) {
	err := analyzeSource(t, `
		const consoleRef = globalThis.console;
		consoleRef.log("ready");
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsUriUtilityGlobals(t *testing.T) {
	err := analyzeSource(t, `
		const encoded = encodeURI("https://example.test/a b");
		const decoded = decodeURI(encoded);
		const part = encodeURIComponent("a b");
		const raw = decodeURIComponent(part);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsGlobalThisRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var globalThis = {};`)
	requireSemanticError(t, err, "duplicate declaration globalThis")
}

func TestSemanticRejectsNewGlobalThis(t *testing.T) {
	err := analyzeSource(t, `const value = new globalThis();`)
	requireSemanticError(t, err, "new target globalThis is not constructable")
}
