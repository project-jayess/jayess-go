package test

import "testing"

func TestSemanticAnalyzesForAwaitOfInAsyncFunction(t *testing.T) {
	err := analyzeSource(t, `
		async function collect(items) {
			for await (const item of items) {
				item;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesForAwaitOfAsyncGeneratorCall(t *testing.T) {
	err := analyzeSource(t, `
		async function* ids() {
			yield 1;
		}
		async function collect() {
			for await (const value of ids()) {
				print(value);
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesAsyncIteratorObjectShape(t *testing.T) {
	err := analyzeSource(t, `
		const asyncIterator = {
			async next() {
				return { value: 1, done: true };
			}
		};
		const values = {
			[Symbol.asyncIterator]() {
				return asyncIterator;
			}
		};
		async function collect() {
			for await (const value of values) {
				print(value);
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsForAwaitOfOutsideAsyncFunction(t *testing.T) {
	err := analyzeSource(t, `
		function collect(items) {
			for await (const item of items) {
				item;
			}
		}
	`)
	requireSemanticError(t, err, "for await outside async function")
}
