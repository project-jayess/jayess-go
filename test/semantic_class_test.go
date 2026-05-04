package test

import "testing"

func TestSemanticDeclaresClassName(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			constructor(value) {
				this.value = value;
			}
			total() {
				return this.value;
			}
		}
		const instance = new Counter(1);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesClassMethodBodies(t *testing.T) {
	err := analyzeSource(t, `
		class Broken {
			value() {
				return missing;
			}
		}
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticRejectsDuplicateClassDeclaration(t *testing.T) {
	err := analyzeSource(t, `
		class Item {}
		var Item = 1;
	`)
	requireSemanticError(t, err, "duplicate declaration Item")
}
