package test

import "testing"

func TestSemanticAnalyzesDeleteTarget(t *testing.T) {
	err := analyzeSource(t, `
		const item = { value: 1 };
		delete item.value;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsDeleteUnknownTarget(t *testing.T) {
	err := analyzeSource(t, `delete missing.value;`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticRejectsDeleteIdentifier(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		delete value;
	`)
	requireSemanticError(t, err, "delete of identifier is not allowed")
}

func TestSemanticRejectsDeletePrivateMember(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			#value = 1;
			clear() {
				delete this.#value;
			}
		}
	`)
	requireSemanticError(t, err, "delete of private member is not allowed")
}
