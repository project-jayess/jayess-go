package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedClassFieldDefiniteAssignment(t *testing.T) {
	_, err := parseProgramError(`class Widget { value!; }`)
	requireDefiniteAssignmentAssertionError(t, err)
}

func TestParserRejectsUnsupportedPrivateClassFieldDefiniteAssignment(t *testing.T) {
	_, err := parseProgramError(`class Widget { #value!; }`)
	requireDefiniteAssignmentAssertionError(t, err)
}

func TestParserRejectsUnsupportedComputedClassFieldDefiniteAssignment(t *testing.T) {
	_, err := parseProgramError(`class Widget { [key]!; }`)
	requireDefiniteAssignmentAssertionError(t, err)
}

func TestParserRejectsUnsupportedStaticClassFieldDefiniteAssignment(t *testing.T) {
	_, err := parseProgramError(`class Widget { static value!; }`)
	requireDefiniteAssignmentAssertionError(t, err)
}

func requireDefiniteAssignmentAssertionError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected definite assignment assertion error")
	}
	if !strings.Contains(err.Error(), "definite assignment assertions are not supported") {
		t.Fatalf("expected unsupported definite assignment diagnostic, got %v", err)
	}
}
