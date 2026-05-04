package test

import "testing"

func TestSemanticAnalyzesArrayIndexMutation(t *testing.T) {
	err := analyzeSource(t, `
		const values = [1, 2];
		const next = 3;
		values[0] = next;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesArrayLengthMemberAccess(t *testing.T) {
	err := analyzeSource(t, `
		const values = [1, 2];
		const count = values.length;
		print(count);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesArrayIterationForOf(t *testing.T) {
	err := analyzeSource(t, `
		const values = [1, 2];
		for (const value of values) {
			print(value);
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownArrayMutationValue(t *testing.T) {
	err := analyzeSource(t, `
		const values = [1, 2];
		values[0] = missing;
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}
