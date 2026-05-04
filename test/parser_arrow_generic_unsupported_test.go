package test

import (
	"strings"
	"testing"
)

func TestParserRejectsGenericArrowTypeParametersWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const identity = <T>(value) => value;`)
	requireGenericArrowError(t, err)
}

func TestParserRejectsAsyncGenericArrowTypeParametersWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const identity = async <T>(value) => value;`)
	requireGenericArrowError(t, err)
}

func TestParserRejectsGenericArrowTypeParametersWithMultipleNames(t *testing.T) {
	_, err := parseProgramError(`const pair = <T, U>(left, right) => left;`)
	requireGenericArrowError(t, err)
}

func requireGenericArrowError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected unsupported generic arrow type parameters error")
	}
	if !strings.Contains(err.Error(), "generic type parameters are not supported") {
		t.Fatalf("expected clear generic type parameter diagnostic, got %v", err)
	}
}
