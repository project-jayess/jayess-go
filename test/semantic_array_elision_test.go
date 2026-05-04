package test

import "testing"

func TestSemanticAllowsArrayLiteralElisions(t *testing.T) {
	err := analyzeSource(t, `
		const first = 1;
		const third = 3;
		const values = [first, , third];
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesArrayLiteralElementsAroundElisions(t *testing.T) {
	err := analyzeSource(t, `const values = [first, , third];`)
	requireSemanticError(t, err, "use of first before declaration")
}
