package test

import "testing"

func TestSemanticAnalyzesPromiseErrorPropagationShape(t *testing.T) {
	err := analyzeSource(t, `
		Promise.resolve(1)
			.then(value => { throw new Error("failed"); })
			.catch(error => error);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownPromiseCatchValue(t *testing.T) {
	err := analyzeSource(t, `
		Promise.resolve(1)
			.catch(error => missing);
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}
