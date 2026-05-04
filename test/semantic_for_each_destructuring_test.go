package test

import "testing"

func TestSemanticAllowsForOfDestructuringBindings(t *testing.T) {
	err := analyzeSource(t, `
		const entries = [[1, 2]];
		for (const [name, count] of entries) {
			print(name);
			print(count);
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsForInDestructuringBindings(t *testing.T) {
	err := analyzeSource(t, `
		const item = { name: "jayess" };
		for (const { name } in item) {
			print(name);
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsDuplicateForOfDestructuringBinding(t *testing.T) {
	err := analyzeSource(t, `
		const entries = [[1, 2]];
		for (const [name, name] of entries) {
			print(name);
		}
	`)
	requireSemanticError(t, err, "duplicate for...of binding")
}

func TestSemanticAnalyzesForOfBindingDefault(t *testing.T) {
	err := analyzeSource(t, `
		const fallback = "value";
		const entries = [{}];
		for (const { name = fallback } of entries) {
			print(name);
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownForOfBindingDefault(t *testing.T) {
	err := analyzeSource(t, `
		const entries = [{}];
		for (const { name = fallback } of entries) {
			print(name);
		}
	`)
	requireSemanticError(t, err, "use of fallback before declaration")
}
