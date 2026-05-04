package test

import "testing"

func TestSemanticAnalyzesCustomIterableShape(t *testing.T) {
	err := analyzeSource(t, `
		const iterator = {
			next() {
				return { value: 1, done: false };
			}
		};
		const values = {
			[Symbol.iterator]() {
				return iterator;
			}
		};
		for (const value of values) {
			print(value);
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesIteratorProtocolShape(t *testing.T) {
	err := analyzeSource(t, `
		const current = 1;
		const iterator = {
			next() {
				return { value: current, done: false };
			}
		};
		iterator.next();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownIteratorProtocolValue(t *testing.T) {
	err := analyzeSource(t, `
		const iterator = {
			next() {
				return { value: missing, done: false };
			}
		};
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticAnalyzesCustomIterableComputedKey(t *testing.T) {
	err := analyzeSource(t, `
		const values = {
			[missing.iterator]() {
				return {};
			}
		};
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}
