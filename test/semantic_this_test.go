package test

import "testing"

func TestSemanticRejectsThisAtTopLevel(t *testing.T) {
	err := analyzeSource(t, `this.value;`)
	requireSemanticError(t, err, "this outside function or method")
}

func TestSemanticAllowsThisInPlainFunction(t *testing.T) {
	err := analyzeSource(t, `
		function value() {
			return this.value;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsThisInNestedPlainFunction(t *testing.T) {
	err := analyzeSource(t, `
		const counter = {
			value() {
				function inner() {
					return this.value;
				}
			}
		};
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
