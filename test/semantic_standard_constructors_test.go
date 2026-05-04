package test

import "testing"

func TestSemanticAllowsStandardCollectionConstructors(t *testing.T) {
	err := analyzeSource(t, `
		const items = new Map();
		const names = new Set();
		const weakItems = new WeakMap();
		const weakNames = new WeakSet();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsStandardOtherConstructors(t *testing.T) {
	err := analyzeSource(t, `
		const values = new Array();
		const today = new Date();
		const matcher = new RegExp("a+");
		const item = new Object();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsStandardConstructorInstanceof(t *testing.T) {
	err := analyzeSource(t, `
		function check(value) {
			return value instanceof Map || value instanceof Date || value instanceof Object;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsStandardConstructorRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var Map = {};`)
	requireSemanticError(t, err, "duplicate declaration Map")
}
