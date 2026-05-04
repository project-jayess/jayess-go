package test

import "testing"

func TestSemanticAllowsObjectCatchDestructuring(t *testing.T) {
	err := analyzeSource(t, `
		function work() {}
		try {
			work();
		} catch ({ message }) {
			message;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsArrayCatchDestructuring(t *testing.T) {
	err := analyzeSource(t, `
		function work() {}
		try {
			work();
		} catch ([code]) {
			code;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsDuplicateCatchDestructuringBinding(t *testing.T) {
	err := analyzeSource(t, `
		function work() {}
		try {
			work();
		} catch ([item, item]) {
			item;
		}
	`)
	requireSemanticError(t, err, "duplicate declaration")
}
