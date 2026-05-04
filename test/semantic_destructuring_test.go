package test

import "testing"

func TestSemanticDeclaresArrayDestructuringNames(t *testing.T) {
	err := analyzeSource(t, `
		const values = [1, 2];
		const [first, second] = values;
		first;
		second;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsArrayDestructuringElisions(t *testing.T) {
	err := analyzeSource(t, `
		const values = [1, 2, 3];
		const [first, , third] = values;
		first;
		third;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticDeclaresObjectDestructuringNames(t *testing.T) {
	err := analyzeSource(t, `
		const item = { name: "Jay", count: 2 };
		const { name, count: total } = item;
		name;
		total;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticDeclaresKeywordObjectDestructuringNames(t *testing.T) {
	err := analyzeSource(t, `
		const item = { default: 1, class: "Widget" };
		const { default: value, class: kind } = item;
		value;
		kind;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticDeclaresComputedObjectDestructuringNames(t *testing.T) {
	err := analyzeSource(t, `
		const key = "name";
		const item = { name: "Jay" };
		const { [key]: value } = item;
		value;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesComputedObjectDestructuringKey(t *testing.T) {
	err := analyzeSource(t, `
		const item = {};
		const { [missing]: value } = item;
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticRejectsDuplicateDestructuringBinding(t *testing.T) {
	err := analyzeSource(t, `
		const values = [1, 2];
		const [item, item] = values;
	`)
	requireSemanticError(t, err, "duplicate declaration")
}

func TestSemanticRejectsConstDestructuringReassignment(t *testing.T) {
	err := analyzeSource(t, `
		const values = [1];
		const [item] = values;
		item = 2;
	`)
	requireSemanticError(t, err, "assignment to const item")
}
