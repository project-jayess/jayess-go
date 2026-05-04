package test

import "testing"

func TestSemanticAllowsPromiseConstructorAndHelpers(t *testing.T) {
	err := analyzeSource(t, `
		const promise = new Promise(resolve => resolve(1));
		const resolved = Promise.resolve(1);
		const combined = Promise.all([promise, resolved]);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsTimerGlobals(t *testing.T) {
	err := analyzeSource(t, `
		const timeout = setTimeout(() => print("tick"), 10);
		clearTimeout(timeout);
		const interval = setInterval(() => print("again"), 10);
		clearInterval(interval);
		queueMicrotask(() => print("micro"));
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsPromiseInstanceof(t *testing.T) {
	err := analyzeSource(t, `
		function check(value) {
			return value instanceof Promise;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsPromiseRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var Promise = {};`)
	requireSemanticError(t, err, "duplicate declaration Promise")
}

func TestSemanticRejectsTimerRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var setTimeout = {};`)
	requireSemanticError(t, err, "duplicate declaration setTimeout")
}
