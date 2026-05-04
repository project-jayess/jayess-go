package test

import "testing"

func TestSemanticAllowsBindingDefaultUsingEarlierName(t *testing.T) {
	err := analyzeSource(t, `
		const fallback = 1;
		const values = [];
		const [first = fallback] = values;
		first;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsBindingDefaultUsingUndeclaredName(t *testing.T) {
	err := analyzeSource(t, `
		const values = [];
		const [first = fallback] = values;
	`)
	requireSemanticError(t, err, "use of fallback before declaration")
}

func TestSemanticRejectsBindingDefaultUsingOwnName(t *testing.T) {
	err := analyzeSource(t, `
		const values = [];
		const [first = first] = values;
	`)
	requireSemanticError(t, err, "use of first before declaration")
}
