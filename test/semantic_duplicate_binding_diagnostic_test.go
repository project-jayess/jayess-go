package test

import "testing"

func TestSemanticReportsDuplicateArrayDeclarationBindingName(t *testing.T) {
	err := analyzeSource(t, `
		const values = [1, 2];
		const [item, item] = values;
	`)
	requireSemanticError(t, err, "duplicate declaration item")
}

func TestSemanticReportsDuplicateObjectDeclarationBindingName(t *testing.T) {
	err := analyzeSource(t, `
		const values = { first: 1, second: 2 };
		const { first: item, second: item } = values;
	`)
	requireSemanticError(t, err, "duplicate declaration item")
}

func TestSemanticReportsDuplicateCatchBindingName(t *testing.T) {
	err := analyzeSource(t, `
		try {
			throw "error";
		} catch ({ first: item, second: item }) {
			item;
		}
	`)
	requireSemanticError(t, err, "duplicate declaration item")
}

func TestSemanticReportsDuplicateForOfBindingName(t *testing.T) {
	err := analyzeSource(t, `
		const values = [[1, 2]];
		for (const [item, item] of values) {
			item;
		}
	`)
	requireSemanticError(t, err, "duplicate for...of binding item")
}

func TestSemanticReportsDuplicateForInBindingName(t *testing.T) {
	err := analyzeSource(t, `
		const values = { first: 1 };
		for (const [item, item] in values) {
			item;
		}
	`)
	requireSemanticError(t, err, "duplicate for...in binding item")
}
