package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedFunctionOverloadDeclaration(t *testing.T) {
	_, err := parseProgramError(`function read(value);`)
	requireFunctionOverloadError(t, err)
}

func TestParserRejectsUnsupportedAsyncFunctionOverloadDeclaration(t *testing.T) {
	_, err := parseProgramError(`async function read(value);`)
	requireFunctionOverloadError(t, err)
}

func requireFunctionOverloadError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected function overload declaration error")
	}
	if !strings.Contains(err.Error(), "function overload declarations are not supported") {
		t.Fatalf("expected unsupported function overload diagnostic, got %v", err)
	}
}
