package test

import "testing"

func TestSemanticAnalyzesInOperatorOperands(t *testing.T) {
	err := analyzeSource(t, `
		const key = "name";
		const item = { name: "Jay" };
		key in item;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownInOperatorOperand(t *testing.T) {
	err := analyzeSource(t, `"name" in item;`)
	requireSemanticError(t, err, "use of item before declaration")
}
