package test

import "testing"

func TestSemanticAllowsForOfAssignmentTarget(t *testing.T) {
	err := analyzeSource(t, `
		var item = 0;
		const items = [1, 2];
		for (item of items) {
			print(item);
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsConstForOfAssignmentTarget(t *testing.T) {
	err := analyzeSource(t, `
		const item = 0;
		const items = [1, 2];
		for (item of items) {
			print(item);
		}
	`)
	requireSemanticError(t, err, "assignment to const item")
}

func TestSemanticAllowsForInIndexAssignmentTarget(t *testing.T) {
	err := analyzeSource(t, `
		const keys = [];
		var index = 0;
		const item = { name: "jayess" };
		for (keys[index] in item) {
			index += 1;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
