package test

import "testing"

func TestSemanticAllowsKeywordObjectPropertyNames(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		const item = { default: value, class: "Widget" };
		item.default;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsKeywordObjectMethodNames(t *testing.T) {
	err := analyzeSource(t, `
		const item = {
			default(value) {
				return value;
			}
		};
		item.default(1);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
