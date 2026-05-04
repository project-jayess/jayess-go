package test

import "testing"

func TestSemanticAllowsClassComputedMembers(t *testing.T) {
	err := analyzeSource(t, `
		const methodName = "read";
		const fieldName = "value";
		class Item {
			[fieldName] = 1;
			[methodName]() {
				return this.value;
			}
			get ["label"]() {
				return this.value;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesClassComputedMemberKeys(t *testing.T) {
	err := analyzeSource(t, `
		class Item {
			[missing]() {
				return 1;
			}
		}
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}
