package test

import "testing"

func TestSemanticAnalyzesYieldOperand(t *testing.T) {
	err := analyzeSource(t, `
		function* ids(value) {
			yield value;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsBareYield(t *testing.T) {
	err := analyzeSource(t, `
		function* ids() {
			yield;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesDelegateYieldOperand(t *testing.T) {
	err := analyzeSource(t, `
		function* ids(values) {
			yield* values;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesGeneratorNextInvocation(t *testing.T) {
	err := analyzeSource(t, `
		function* ids() {
			yield 1;
		}
		const first = ids().next();
		print(first);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesForOfGeneratorCall(t *testing.T) {
	err := analyzeSource(t, `
		function* ids() {
			yield 1;
		}
		for (const value of ids()) {
			print(value);
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownDelegateYieldOperand(t *testing.T) {
	err := analyzeSource(t, `
		function* ids() {
			yield* missing;
		}
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticRejectsUnknownYieldOperand(t *testing.T) {
	err := analyzeSource(t, `
		function* ids() {
			yield missing;
		}
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticRejectsYieldOutsideGeneratorFunction(t *testing.T) {
	err := analyzeSource(t, `
		function ids(value) {
			yield value;
		}
	`)
	requireSemanticError(t, err, "yield outside generator function")
}

func TestSemanticRejectsYieldInsideNestedNonGeneratorFunction(t *testing.T) {
	err := analyzeSource(t, `
		function* outer(value) {
			function inner() {
				yield value;
			}
		}
	`)
	requireSemanticError(t, err, "yield outside generator function")
}

func TestSemanticAnalyzesYieldInSwitchDiscriminantAndCase(t *testing.T) {
	err := analyzeSource(t, `
		function* run(kind, expected) {
			switch (yield kind) {
			case yield expected:
				break;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesAsyncGeneratorFunctionExpression(t *testing.T) {
	err := analyzeSource(t, `
		const ids = async function* (next) {
			yield await next();
		};
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownAwaitInAsyncGeneratorExpression(t *testing.T) {
	err := analyzeSource(t, `
		const ids = async function* () {
			yield await missing;
		};
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}
