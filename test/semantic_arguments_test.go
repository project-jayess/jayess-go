package test

import "testing"

func TestSemanticAllowsArgumentsInFunction(t *testing.T) {
	err := analyzeSource(t, `
		function first() {
			return arguments;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsArgumentsInObjectMethod(t *testing.T) {
	err := analyzeSource(t, `
		const item = {
			read() {
				return arguments;
			}
		};
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsArrowToUseOuterArguments(t *testing.T) {
	err := analyzeSource(t, `
		function outer() {
			const read = () => arguments;
			return read;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsNestedFunctionOwnArguments(t *testing.T) {
	err := analyzeSource(t, `
		function outer() {
			function inner() {
				return arguments;
			}
			return arguments;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsTopLevelArrowArguments(t *testing.T) {
	err := analyzeSource(t, `const read = () => arguments;`)
	requireSemanticError(t, err, "use of arguments before declaration")
}
