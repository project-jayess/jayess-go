package test

import "testing"

func TestSemanticAnalyzesUnaryPlusOperand(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		const copy = +value;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownUnaryPlusOperand(t *testing.T) {
	err := analyzeSource(t, `const copy = +missing;`)
	requireSemanticError(t, err, "use of missing before declaration")
}
