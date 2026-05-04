package test

import "testing"

func TestSemanticAllowsPromiseResolveRejectAndChaining(t *testing.T) {
	err := analyzeSource(t, `
		const resolved = Promise.resolve(1);
		const rejected = Promise.reject(new Error("failed"));
		resolved
			.then(value => value)
			.catch(error => rejected)
			.finally(() => print("done"));
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsPromiseCombinators(t *testing.T) {
	err := analyzeSource(t, `
		const first = Promise.resolve(1);
		const second = Promise.resolve(2);
		Promise.all([first, second]);
		Promise.race([first, second]);
		Promise.allSettled([first, second]);
		Promise.any([first, second]);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
