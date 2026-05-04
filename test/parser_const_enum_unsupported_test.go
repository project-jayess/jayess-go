package test

import (
	"strings"
	"testing"
)

func TestParserRejectsConstEnumDeclarationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const enum Color { Red, Blue }`)
	requireConstEnumError(t, err)
}

func TestParserRejectsExportedConstEnumDeclarationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`export const enum Color { Red, Blue }`)
	requireConstEnumError(t, err)
}

func requireConstEnumError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected unsupported const enum declaration error")
	}
	if !strings.Contains(err.Error(), "enum declarations are not supported") {
		t.Fatalf("expected clear enum diagnostic, got %v", err)
	}
}
