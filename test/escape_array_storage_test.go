package test

import (
	"testing"

	"jayess-go/escape"
)

func TestEscapeMarksArrayElementValueAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			const array = [value];
			return array;
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("value") {
		t.Fatalf("expected array element value to escape")
	}
}

func TestEscapeMarksArrayElementExpressionValuesAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function make(condition) {
			const first = {};
			const second = {};
			const array = [condition ? first : second];
			return array;
		}
	`)

	report := escape.Analyze(program)
	for _, name := range []string{"condition", "first", "second"} {
		if !report.Escapes(name) {
			t.Fatalf("expected array element expression identifier %s to escape", name)
		}
	}
}

func TestEscapeMarksSpreadArrayElementAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const values = [];
			const array = [...values];
			return array;
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("values") {
		t.Fatalf("expected spread array element to escape")
	}
}
