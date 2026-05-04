package test

import (
	"testing"

	"jayess-go/escape"
)

func TestEscapeMarksClosureCapturedVariableAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			return () => value;
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("value") {
		t.Fatalf("expected closure-captured value to escape")
	}
}

func TestEscapeMarksClosureCapturedParameterAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function make(value) {
			return () => value;
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("value") {
		t.Fatalf("expected closure-captured parameter to escape")
	}
}

func TestEscapeDoesNotMarkClosureLocalAsCaptured(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			return () => {
				const local = {};
				return local;
			};
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("local") {
		t.Fatalf("expected returned closure-local value to escape")
	}
	if report.Escapes("make") {
		t.Fatalf("did not expect function name to escape")
	}
}
