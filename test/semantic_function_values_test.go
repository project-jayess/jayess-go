package test

import "testing"

func TestSemanticAllowsFunctionDeclarationRecursion(t *testing.T) {
	err := analyzeSource(t, `
		function repeat(value) {
			return repeat(value);
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsNamedFunctionExpressionRecursion(t *testing.T) {
	err := analyzeSource(t, `
		const repeat = function again(value) {
			return again(value);
		};
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsFunctionValuesInVariablesArraysAndObjects(t *testing.T) {
	err := analyzeSource(t, `
		const identity = function (value) {
			return value;
		};
		const callbacks = [identity, function (value) { return value; }];
		const tools = { identity: identity };
		identity(1);
		callbacks[0](2);
		tools.identity(3);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsHigherOrderCallbacks(t *testing.T) {
	err := analyzeSource(t, `
		function apply(callback, value) {
			return callback(value);
		}
		const result = apply(function (item) {
			return item;
		}, 1);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesReturnedClosureCapturingOuterVariable(t *testing.T) {
	err := analyzeSource(t, `
		function makeAdder(base) {
			return function (value) {
				return base + value;
			};
		}
		const addOne = makeAdder(1);
		addOne(2);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownClosureCapturedVariable(t *testing.T) {
	err := analyzeSource(t, `
		function makeAdder() {
			return function (value) {
				return missing + value;
			};
		}
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}
