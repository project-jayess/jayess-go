package test

import "testing"

func TestSemanticAnalyzesAwaitOperand(t *testing.T) {
	err := analyzeSource(t, `
		async function load(read) {
			return await read();
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesAsyncReturnValue(t *testing.T) {
	err := analyzeSource(t, `
		async function load(value) {
			return value + 1;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownAsyncReturnValue(t *testing.T) {
	err := analyzeSource(t, `
		async function load() {
			return missing;
		}
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticRejectsUnknownAwaitOperand(t *testing.T) {
	err := analyzeSource(t, `
		async function load() {
			return await missing;
		}
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticRejectsAwaitOutsideAsyncFunction(t *testing.T) {
	err := analyzeSource(t, `
		function load(read) {
			return await read();
		}
	`)
	requireSemanticError(t, err, "await outside async function")
}

func TestSemanticRejectsAwaitInsideNestedNonAsyncFunction(t *testing.T) {
	err := analyzeSource(t, `
		async function outer(read) {
			function inner() {
				return await read();
			}
		}
	`)
	requireSemanticError(t, err, "await outside async function")
}

func TestSemanticAnalyzesAwaitInForCondition(t *testing.T) {
	err := analyzeSource(t, `
		async function run(ready) {
			for (; await ready; ) {
				break;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesAwaitInForInObject(t *testing.T) {
	err := analyzeSource(t, `
		async function run(object) {
			for (const key in await object) {
				key;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesAwaitInSwitchDiscriminantAndCase(t *testing.T) {
	err := analyzeSource(t, `
		async function run(kind, expected) {
			switch (await kind) {
			case await expected:
				break;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
