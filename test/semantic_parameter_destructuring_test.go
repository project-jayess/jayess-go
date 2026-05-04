package test

import "testing"

func TestSemanticAllowsDestructuringParameters(t *testing.T) {
	err := analyzeSource(t, `
		function describe({ name }, [count]) {
			return name + count;
		}
		describe({ name: "Jay" }, [1]);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsDestructuringParameterDefaults(t *testing.T) {
	err := analyzeSource(t, `
		const fallback = "Jay";
		function describe({ name = fallback }) {
			return name;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsDuplicateDestructuringParameter(t *testing.T) {
	err := analyzeSource(t, `
		function bad({ name }, name) {
			return name;
		}
	`)
	requireSemanticError(t, err, "duplicate parameter")
}

func TestSemanticRejectsUnknownDestructuringParameterDefault(t *testing.T) {
	err := analyzeSource(t, `
		function describe({ name = fallback }) {
			return name;
		}
	`)
	requireSemanticError(t, err, "use of fallback before declaration")
}
