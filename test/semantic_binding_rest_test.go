package test

import "testing"

func TestSemanticDeclaresArrayRestBinding(t *testing.T) {
	err := analyzeSource(t, `
		const values = [1, 2, 3];
		const [first, ...rest] = values;
		first;
		rest;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsDuplicateRestBinding(t *testing.T) {
	err := analyzeSource(t, `
		const values = [1, 2, 3];
		const [rest, ...rest] = values;
	`)
	requireSemanticError(t, err, "duplicate declaration")
}
