package test

import "testing"

func TestSemanticAllowsArrowThisInObjectMethod(t *testing.T) {
	err := analyzeSource(t, `
		const counter = {
			value: 1,
			read() {
				const getValue = () => this.value;
				return getValue();
			}
		};
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsArrowThisInClassMethod(t *testing.T) {
	err := analyzeSource(t, `
		class Counter {
			#value = 1;
			read() {
				const getValue = () => this.#value;
				return getValue();
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsArrowThisOutsideMethod(t *testing.T) {
	err := analyzeSource(t, `
		const getValue = () => this.value;
	`)
	requireSemanticError(t, err, "this outside function or method")
}

func TestSemanticAllowsArrowThisInPlainFunction(t *testing.T) {
	err := analyzeSource(t, `
		function value() {
			const getValue = () => this.value;
			return getValue();
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
