package test

import "testing"

func TestSemanticAllowsUpdateExpressions(t *testing.T) {
	err := analyzeSource(t, `
		var count = 1;
		count++;
		--count;
		const item = { count: 1 };
		item.count++;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsConstUpdateExpression(t *testing.T) {
	err := analyzeSource(t, `
		const count = 1;
		count++;
	`)
	requireSemanticError(t, err, "assignment to const count")
}

func TestSemanticRejectsUnknownUpdateTarget(t *testing.T) {
	err := analyzeSource(t, `missing++;`)
	requireSemanticError(t, err, "assignment to missing before declaration")
}
