package test

import "testing"

func TestSemanticAllowsVarReassignment(t *testing.T) {
	err := analyzeSource(t, `
		var value = 1;
		value = 2;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsConstReassignment(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		value = 2;
	`)
	requireSemanticError(t, err, "assignment to const value")
}

func TestSemanticAllowsConstObjectPropertyAssignment(t *testing.T) {
	err := analyzeSource(t, `
		const item = { count: 1 };
		item.count = 2;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsConstForOfReassignment(t *testing.T) {
	err := analyzeSource(t, `
		const items = [1, 2];
		for (const item of items) {
			item = 3;
		}
	`)
	requireSemanticError(t, err, "assignment to const item")
}
