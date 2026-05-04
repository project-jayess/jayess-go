package test

import "testing"

func TestSemanticAllowsBuiltinErrorConstructors(t *testing.T) {
	err := analyzeSource(t, `
		function fail(message) {
			throw new Error(message);
		}
		function badType(message) {
			throw new TypeError(message);
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsAdditionalBuiltinErrorConstructors(t *testing.T) {
	err := analyzeSource(t, `
		function aggregate(errors) {
			return new AggregateError(errors);
		}
		function evalProblem(message) {
			return new EvalError(message);
		}
		function uriProblem(message) {
			return new URIError(message);
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsBuiltinErrorInstanceof(t *testing.T) {
	err := analyzeSource(t, `
		function check(error) {
			return error instanceof RangeError;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsAdditionalBuiltinErrorInstanceof(t *testing.T) {
	err := analyzeSource(t, `
		function check(error) {
			return error instanceof EvalError || error instanceof URIError;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsBuiltinErrorRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var Error = {};`)
	requireSemanticError(t, err, "duplicate declaration Error")
}

func TestSemanticRejectsAdditionalBuiltinErrorRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var URIError = {};`)
	requireSemanticError(t, err, "duplicate declaration URIError")
}
