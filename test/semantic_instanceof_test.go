package test

import "testing"

func TestSemanticAllowsInstanceofWithClassTarget(t *testing.T) {
	err := analyzeSource(t, `
		class Widget {}
		const value = new Widget();
		const ok = value instanceof Widget;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsInstanceofWithFunctionTarget(t *testing.T) {
	err := analyzeSource(t, `
		function Widget() {
			return 1;
		}
		const value = {};
		const ok = value instanceof Widget;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsInstanceofWithVariableTarget(t *testing.T) {
	err := analyzeSource(t, `
		const Widget = {};
		const value = {};
		const ok = value instanceof Widget;
	`)
	requireSemanticError(t, err, "instanceof target Widget is not constructable")
}

func TestSemanticAllowsInstanceofWithImportedTarget(t *testing.T) {
	err := analyzeSource(t, `
		import { Widget } from "./widget.js";
		const value = {};
		const ok = value instanceof Widget;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
