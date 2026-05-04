package test

import "testing"

func TestSemanticAnalyzesVoidOperand(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		const ignored = void value;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownVoidOperand(t *testing.T) {
	err := analyzeSource(t, `const ignored = void missing;`)
	requireSemanticError(t, err, "use of missing before declaration")
}
