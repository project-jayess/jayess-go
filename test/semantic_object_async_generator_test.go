package test

import "testing"

func TestSemanticAnalyzesObjectAsyncMethod(t *testing.T) {
	err := analyzeSource(t, `
		const loader = {
			async load() {
				await 1;
			}
		};
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesObjectGeneratorMethod(t *testing.T) {
	err := analyzeSource(t, `
		const ids = {
			*values() {
				yield 1;
			}
		};
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsAwaitInNonAsyncObjectMethod(t *testing.T) {
	err := analyzeSource(t, `
		const loader = {
			load() {
				await 1;
			}
		};
	`)
	requireSemanticError(t, err, "await outside async function")
}
