package test

import "testing"

func TestSemanticAnalyzesClassFieldInitializers(t *testing.T) {
	err := analyzeSource(t, `
		const initial = 1;
		class Counter {
			value = initial;
			static total = initial;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesThisInClassFieldInitializer(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			seed = 1;
			value = this.seed;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesPrivateAccessInClassFieldInitializer(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			#seed = 1;
			value = this.#seed;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownClassFieldInitializerName(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			value = missing;
		}
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticRejectsUnknownPrivateClassFieldInitializerAccess(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			value = this.#missing;
		}
	`)
	requireSemanticError(t, err, "private member #missing is not declared")
}
