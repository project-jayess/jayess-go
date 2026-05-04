package test

import (
	"testing"

	"jayess-go/escape"
)

func TestEscapeMarksReturnedIdentifierAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			return value;
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("value") {
		t.Fatalf("expected returned value to escape")
	}
}

func TestEscapeMarksReturnedExpressionIdentifiersAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function make(condition) {
			const first = {};
			const second = {};
			return condition ? first : second;
		}
	`)

	report := escape.Analyze(program)
	for _, name := range []string{"condition", "first", "second"} {
		if !report.Escapes(name) {
			t.Fatalf("expected returned expression identifier %s to escape", name)
		}
	}
}

func TestEscapeDoesNotMarkUnreturnedLocalAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			return 1;
		}
	`)

	report := escape.Analyze(program)
	if report.Escapes("value") {
		t.Fatalf("did not expect unreturned local to escape")
	}
}
