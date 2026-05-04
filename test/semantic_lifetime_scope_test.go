package test

import "testing"

func TestSemanticRejectsFunctionLocalOutsideFunctionScope(t *testing.T) {
	err := analyzeSource(t, `
		function make() {
			const local = 1;
			return local;
		}
		local;
	`)
	requireSemanticError(t, err, "use of local before declaration")
}

func TestSemanticRejectsLoopLocalOutsideLoopScope(t *testing.T) {
	err := analyzeSource(t, `
		for (var index = 0; index < 1; index += 1) {
			const local = index;
		}
		index;
	`)
	requireSemanticError(t, err, "use of index before declaration")
}

func TestSemanticRejectsCatchBindingOutsideCatchScope(t *testing.T) {
	err := analyzeSource(t, `
		const failure = new Error("failed");
		try {
			throw failure;
		} catch (error) {
			print(error);
		}
		error;
	`)
	requireSemanticError(t, err, "use of error before declaration")
}

func TestSemanticAllowsModuleValuesInsideFunctionScopes(t *testing.T) {
	err := analyzeSource(t, `
		const moduleValue = 1;
		function readModuleValue() {
			return moduleValue;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsModuleValuesInsideNestedBlockScopes(t *testing.T) {
	err := analyzeSource(t, `
		var moduleValue = 1;
		{
			{
				print(moduleValue);
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
