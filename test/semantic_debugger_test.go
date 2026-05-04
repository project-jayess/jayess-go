package test

import "testing"

func TestSemanticAllowsDebuggerStatement(t *testing.T) {
	err := analyzeSource(t, `
		function main() {
			debugger;
			return 0;
		}
	`)
	if err != nil {
		t.Fatalf("expected debugger statement to analyze, got %v", err)
	}
}
