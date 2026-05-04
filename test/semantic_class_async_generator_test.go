package test

import "testing"

func TestSemanticAnalyzesClassAsyncMethod(t *testing.T) {
	err := analyzeSource(t, `
		class Loader {
			async load() {
				await 1;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesClassGeneratorMethod(t *testing.T) {
	err := analyzeSource(t, `
		class Ids {
			*values() {
				yield 1;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsAsyncClassConstructor(t *testing.T) {
	err := analyzeSource(t, `
		class Loader {
			async constructor() {}
		}
	`)
	requireSemanticError(t, err, "constructor cannot be async or generator")
}
