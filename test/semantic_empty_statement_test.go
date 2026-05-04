package test

import "testing"

func TestSemanticIgnoresEmptyStatements(t *testing.T) {
	err := analyzeSource(t, `
		;
		const value = 1;
		{
			;
			value;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
