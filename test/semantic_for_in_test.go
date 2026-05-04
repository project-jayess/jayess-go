package test

import "testing"

func TestSemanticAllowsForInStatement(t *testing.T) {
	err := analyzeSource(t, `
		const item = { name: "jayess" };
		for (const key in item) {
			print(key);
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownForInObject(t *testing.T) {
	err := analyzeSource(t, `
		for (const key in missing) {
			print(key);
		}
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticRejectsConstForInReassignment(t *testing.T) {
	err := analyzeSource(t, `
		const item = { name: "jayess" };
		for (const key in item) {
			key = "next";
		}
	`)
	requireSemanticError(t, err, "assignment to const key")
}

func TestSemanticAllowsLabeledContinueToForIn(t *testing.T) {
	err := analyzeSource(t, `
		const item = { name: "jayess" };
		outer: for (var key in item) {
			continue outer;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
