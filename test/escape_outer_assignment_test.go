package test

import (
	"testing"

	"jayess-go/escape"
)

func TestEscapeMarksValueAssignedToOuterScopeAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function writeOuter() {
			var outer = null;
			{
				const value = {};
				outer = value;
			}
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("value") {
		t.Fatalf("expected value assigned to outer scope to escape")
	}
}

func TestEscapeDoesNotMarkValueAssignedToLocalScopeAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function writeLocal() {
			var local = null;
			const value = {};
			local = value;
		}
	`)

	report := escape.Analyze(program)
	if report.Escapes("value") {
		t.Fatalf("did not expect value assigned to local scope to escape")
	}
}

func TestEscapeMarksExpressionAssignedToOuterScopeAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function writeOuter(condition) {
			var outer = null;
			{
				const first = {};
				const second = {};
				outer = condition ? first : second;
			}
		}
	`)

	report := escape.Analyze(program)
	for _, name := range []string{"condition", "first", "second"} {
		if !report.Escapes(name) {
			t.Fatalf("expected %s assigned to outer scope to escape", name)
		}
	}
}
