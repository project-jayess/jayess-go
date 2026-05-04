package test

import (
	"testing"

	"jayess-go/escape"
)

func TestEscapeClassifiesNonEscapingValueAsCleanupEligible(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			return 1;
		}
	`)

	report := escape.Analyze(program)
	if !report.EligibleForScopeCleanup("value") {
		t.Fatalf("expected non-escaping value to be eligible for scope cleanup")
	}
}

func TestEscapeDoesNotClassifyEscapingValueAsCleanupEligible(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			return value;
		}
	`)

	report := escape.Analyze(program)
	if report.EligibleForScopeCleanup("value") {
		t.Fatalf("did not expect escaping value to be eligible for scope cleanup")
	}
}

func TestEscapeClassifiesReturnedValueAsSurvivingScopeExit(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			return value;
		}
	`)

	report := escape.Analyze(program)
	if !report.MustSurviveScopeExit("value") {
		t.Fatalf("expected returned value to survive scope exit")
	}
}

func TestEscapeClassifiesUnknownCallArgumentAsSurvivingScopeExit(t *testing.T) {
	program := parseProgram(t, `
		function run() {
			const value = {};
			external(value);
		}
	`)

	report := escape.Analyze(program)
	if !report.MustSurviveScopeExit("value") {
		t.Fatalf("expected unknown call argument to survive scope exit")
	}
}
