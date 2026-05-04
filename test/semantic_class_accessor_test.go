package test

import "testing"

func TestSemanticAnalyzesClassAccessorBodies(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			get value() {
				return this.count;
			}
			set value(next) {
				this.count = next;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownNameInClassAccessor(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			get value() {
				return missing;
			}
		}
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}
