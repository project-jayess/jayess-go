package test

import (
	"testing"

	"jayess-go/escape"
)

func TestEscapeMarksObjectShorthandValueAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			const object = { value };
			return object;
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("value") {
		t.Fatalf("expected object shorthand value to escape")
	}
}

func TestEscapeMarksObjectPropertyValueAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			const object = { item: value };
			return object;
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("value") {
		t.Fatalf("expected object property value to escape")
	}
}

func TestEscapeDoesNotMarkComputedObjectKeyAsStoredValue(t *testing.T) {
	program := parseProgram(t, `
		function make(key) {
			const value = {};
			const object = { [key]: value };
			return object;
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("value") {
		t.Fatalf("expected computed object property value to escape")
	}
	if report.Escapes("key") {
		t.Fatalf("did not expect computed object key to escape as stored value")
	}
}
