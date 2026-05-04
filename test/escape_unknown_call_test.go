package test

import (
	"testing"

	"jayess-go/escape"
)

func TestEscapeMarksUnknownCallArgumentsAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function run() {
			const value = {};
			external(value);
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("value") {
		t.Fatalf("expected unknown call argument to escape")
	}
}

func TestEscapeDoesNotMarkKnownDirectCallArgumentsAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function helper(value) {
			return 0;
		}

		function run() {
			const value = {};
			helper(value);
		}
	`)

	report := escape.Analyze(program)
	if report.Escapes("value") {
		t.Fatalf("did not expect known direct call argument to escape")
	}
}

func TestEscapeMarksInvokeArgumentsAsEscaping(t *testing.T) {
	program := parseProgram(t, `
		function run(api) {
			const value = {};
			api.send(value);
		}
	`)

	report := escape.Analyze(program)
	if !report.Escapes("value") {
		t.Fatalf("expected invoke argument to escape")
	}
}
