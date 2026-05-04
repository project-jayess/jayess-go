package test

import "testing"

func TestSemanticAllowsObjectMethodDefinitions(t *testing.T) {
	err := analyzeSource(t, `
		const tools = {
			identity(value) {
				return value;
			}
		};
		tools.identity(1);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsThisInObjectMethod(t *testing.T) {
	err := analyzeSource(t, `
		const counter = {
			value: 1,
			next() {
				return this.value;
			}
		};
		counter.next();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesObjectMethodBody(t *testing.T) {
	err := analyzeSource(t, `
		const tools = {
			broken() {
				return missing;
			}
		};
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}
