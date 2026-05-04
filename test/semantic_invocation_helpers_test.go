package test

import "testing"

func TestSemanticAnalyzesBindInvocationHelper(t *testing.T) {
	err := analyzeSource(t, `
		function identity(value) {
			return value;
		}
		const bound = identity.bind(null, 1);
		bound();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesCallAndApplyInvocationHelpers(t *testing.T) {
	err := analyzeSource(t, `
		function identity(value) {
			return value;
		}
		const args = [1];
		identity.call(null, 1);
		identity.apply(null, args);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesInvocationHelperArguments(t *testing.T) {
	err := analyzeSource(t, `
		function identity(value) {
			return value;
		}
		identity.call(null, missing);
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}
