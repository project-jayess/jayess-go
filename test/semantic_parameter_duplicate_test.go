package test

import "testing"

func TestSemanticRejectsDuplicateNamedParameter(t *testing.T) {
	err := analyzeSource(t, `
		function bad(value, value) {
			return value;
		}
	`)
	requireSemanticError(t, err, "duplicate parameter value")
}

func TestSemanticRejectsDuplicateDefaultParameter(t *testing.T) {
	err := analyzeSource(t, `
		function bad(value, value = 1) {
			return value;
		}
	`)
	requireSemanticError(t, err, "duplicate parameter value")
}

func TestSemanticRejectsDuplicateRestParameter(t *testing.T) {
	err := analyzeSource(t, `
		function bad(value, ...value) {
			return value;
		}
	`)
	requireSemanticError(t, err, "duplicate parameter value")
}

func TestSemanticRejectsDuplicateArrayParameterBinding(t *testing.T) {
	err := analyzeSource(t, `
		function bad([value, value]) {
			return value;
		}
	`)
	requireSemanticError(t, err, "duplicate parameter")
}

func TestSemanticRejectsDuplicateObjectParameterBinding(t *testing.T) {
	err := analyzeSource(t, `
		function bad({ first: value, second: value }) {
			return value;
		}
	`)
	requireSemanticError(t, err, "duplicate parameter")
}
