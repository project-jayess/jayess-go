package test

import "testing"

func TestSemanticAllowsObjectAccessorBodies(t *testing.T) {
	err := analyzeSource(t, `
		const counter = {
			count: 1,
			get value() {
				return this.count;
			},
			set value(next) {
				this.count = next;
			}
		};
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesObjectAccessorBody(t *testing.T) {
	err := analyzeSource(t, `
		const item = {
			get value() {
				return missing;
			}
		};
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticAllowsComputedObjectAccessorBodies(t *testing.T) {
	err := analyzeSource(t, `
		const key = "value";
		const counter = {
			count: 1,
			get [key]() {
				return this.count;
			},
			set [key](next) {
				this.count = next;
			}
		};
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesComputedObjectAccessorKey(t *testing.T) {
	err := analyzeSource(t, `
		const item = {
			get [missing]() {
				return 1;
			}
		};
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}
