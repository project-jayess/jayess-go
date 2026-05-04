package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedVarDefiniteAssignmentAssertion(t *testing.T) {
	_, err := parseProgramError(`var value!;`)
	requireVariableDefiniteAssignmentError(t, err)
}

func TestParserRejectsUnsupportedConstDefiniteAssignmentAssertion(t *testing.T) {
	_, err := parseProgramError(`const value!: number;`)
	requireVariableDefiniteAssignmentError(t, err)
}

func TestParserRejectsUnsupportedExportedDefiniteAssignmentAssertion(t *testing.T) {
	_, err := parseProgramError(`export var value!;`)
	requireVariableDefiniteAssignmentError(t, err)
}

func TestParserRejectsUnsupportedArrayBindingDefiniteAssignmentAssertion(t *testing.T) {
	_, err := parseProgramError(`const [value!] = values;`)
	requireVariableDefiniteAssignmentError(t, err)
}

func TestParserRejectsUnsupportedObjectBindingDefiniteAssignmentAssertion(t *testing.T) {
	_, err := parseProgramError(`const { value! } = item;`)
	requireVariableDefiniteAssignmentError(t, err)
}

func TestParserRejectsUnsupportedNestedObjectBindingDefiniteAssignmentAssertion(t *testing.T) {
	_, err := parseProgramError(`const { value: local! } = item;`)
	requireVariableDefiniteAssignmentError(t, err)
}

func TestParserRejectsUnsupportedRestArrayBindingDefiniteAssignmentAssertion(t *testing.T) {
	_, err := parseProgramError(`const [...rest!] = values;`)
	requireVariableDefiniteAssignmentError(t, err)
}

func TestParserRejectsUnsupportedRestObjectBindingDefiniteAssignmentAssertion(t *testing.T) {
	_, err := parseProgramError(`const { ...rest! } = item;`)
	requireVariableDefiniteAssignmentError(t, err)
}

func requireVariableDefiniteAssignmentError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected definite assignment assertion error")
	}
	if !strings.Contains(err.Error(), "definite assignment assertions are not supported") {
		t.Fatalf("expected definite assignment diagnostic, got %v", err)
	}
}
