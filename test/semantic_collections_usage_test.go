package test

import "testing"

func TestSemanticAnalyzesMapConstructionAndMethodAccess(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		const items = new Map();
		items.set("value", value);
		items.get("value");
		items.has("value");
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesSetConstructionAndMethodAccess(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		const items = new Set();
		items.add(value);
		items.has(value);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesWeakCollectionConstruction(t *testing.T) {
	err := analyzeSource(t, `
		const key = {};
		const values = new WeakMap();
		const keys = new WeakSet();
		values.set(key, 1);
		keys.add(key);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownCollectionMethodOperand(t *testing.T) {
	err := analyzeSource(t, `
		const items = new Map();
		items.set("value", missing);
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}
