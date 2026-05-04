package test

import "testing"

func TestSemanticAllowsTypeofUndeclaredIdentifier(t *testing.T) {
	err := analyzeSource(t, `const kind = typeof missing;`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesTypeofMemberTarget(t *testing.T) {
	err := analyzeSource(t, `const kind = typeof missing.value;`)
	requireSemanticError(t, err, "use of missing before declaration")
}
