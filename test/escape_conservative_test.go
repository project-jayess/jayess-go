package test

import (
	"testing"

	"jayess-go/escape"
)

func TestEscapeConservativelyAnalyzesNestedUnknownCalls(t *testing.T) {
	program := parseProgram(t, `
		function run(condition) {
			const value = {};
			condition && external(value);
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("value") {
		t.Fatalf("expected nested unknown call argument to escape")
	}
}

func TestEscapeConservativelyAnalyzesNestedArrayStorage(t *testing.T) {
	program := parseProgram(t, `
		function run(condition) {
			const value = {};
			const stored = condition ? [value] : [];
			return stored;
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("value") {
		t.Fatalf("expected nested array-stored value to escape")
	}
}

func TestEscapeConservativelyMarksConstructorArguments(t *testing.T) {
	program := parseProgram(t, `
		function run(Factory) {
			const value = {};
			const instance = new Factory(value);
			return instance;
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("value") {
		t.Fatalf("expected constructor argument to escape")
	}
}
