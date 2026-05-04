package test

import "testing"

func TestSemanticAllowsDocumentedBuiltins(t *testing.T) {
	err := analyzeSource(t, `
		function main(args) {
			const delay = 500;
			var total = 10.5 + 2 * 3;
			console.log(total);
			console.log(args[0]);
			sleep(delay);
			var name = readLine("What is your name? ");
			console.log(name);
			readKey("Press any key to continue");
			print(name);
			return 0;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsNumericGlobalConstants(t *testing.T) {
	err := analyzeSource(t, `
		function main() {
			return NaN + Infinity;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsNumericGlobalHelpers(t *testing.T) {
	err := analyzeSource(t, `
		function main(value) {
			const whole = parseInt(value);
			const precise = parseFloat(value);
			return isNaN(whole) || isFinite(precise);
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsStandardNamespaceGlobals(t *testing.T) {
	err := analyzeSource(t, `
		function main(text) {
			const value = JSON.parse(text);
			return Math.max(value.count, 0);
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsTopLevelBuiltinRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var console = {};`)
	requireSemanticError(t, err, "duplicate declaration console")
}

func TestSemanticRejectsTopLevelNumericHelperRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var parseInt = {};`)
	requireSemanticError(t, err, "duplicate declaration parseInt")
}

func TestSemanticRejectsTopLevelNamespaceRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var Math = {};`)
	requireSemanticError(t, err, "duplicate declaration Math")
}

func TestSemanticRejectsNewNamespaceGlobal(t *testing.T) {
	err := analyzeSource(t, `const value = new JSON();`)
	requireSemanticError(t, err, "new target JSON is not constructable")
}

func TestSemanticRejectsTopLevelNumericConstantRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var NaN = 0;`)
	requireSemanticError(t, err, "duplicate declaration NaN")
}

func TestSemanticRejectsNumericConstantAssignment(t *testing.T) {
	err := analyzeSource(t, `Infinity = 1;`)
	requireSemanticError(t, err, "assignment to const Infinity")
}

func TestSemanticAllowsNestedBuiltinShadowing(t *testing.T) {
	err := analyzeSource(t, `
		function main() {
			var console = {};
			console.log;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
