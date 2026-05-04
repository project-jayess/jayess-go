package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedExportEqualsDeclaration(t *testing.T) {
	_, err := parseProgramError(`export = Widget;`)
	requireExportEqualsError(t, err)
}

func TestParserRejectsUnsupportedExportEqualsObjectDeclaration(t *testing.T) {
	_, err := parseProgramError(`export = { Widget };`)
	requireExportEqualsError(t, err)
}

func requireExportEqualsError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected export equals declaration error")
	}
	if !strings.Contains(err.Error(), "export equals declarations are not supported") {
		t.Fatalf("expected unsupported export equals diagnostic, got %v", err)
	}
}
