package test

import "testing"

func TestSemanticDeclaresObjectRestBinding(t *testing.T) {
	err := analyzeSource(t, `
		const item = { name: "Jay", count: 2 };
		const { name, ...rest } = item;
		name;
		rest;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsDuplicateObjectRestBinding(t *testing.T) {
	err := analyzeSource(t, `
		const item = { name: "Jay" };
		const { rest, ...rest } = item;
	`)
	requireSemanticError(t, err, "duplicate declaration")
}
