package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedConstAssertion(t *testing.T) {
	_, err := parseProgramError(`const values = [1, 2] as const;`)
	requireConstAssertionError(t, err)
}

func TestParserRejectsUnsupportedObjectConstAssertion(t *testing.T) {
	_, err := parseProgramError(`const value = { name: "Jayess" } as const;`)
	requireConstAssertionError(t, err)
}

func TestParserStillRejectsOtherAsAssertionsAsTypeAssertions(t *testing.T) {
	_, err := parseProgramError(`const value = item as Widget;`)
	if err == nil {
		t.Fatalf("expected type assertion error")
	}
	if !strings.Contains(err.Error(), "type assertions are not supported") {
		t.Fatalf("expected type assertion diagnostic, got %v", err)
	}
}

func requireConstAssertionError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected const assertion error")
	}
	if !strings.Contains(err.Error(), "const assertions are not supported") {
		t.Fatalf("expected unsupported const assertion diagnostic, got %v", err)
	}
}
