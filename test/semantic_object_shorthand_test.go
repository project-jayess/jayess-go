package test

import "testing"

func TestSemanticAnalyzesObjectShorthandProperty(t *testing.T) {
	err := analyzeSource(t, `
		const name = "Jayess";
		const item = { name };
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownObjectShorthandProperty(t *testing.T) {
	err := analyzeSource(t, `const item = { missing };`)
	requireSemanticError(t, err, "use of missing before declaration")
}
