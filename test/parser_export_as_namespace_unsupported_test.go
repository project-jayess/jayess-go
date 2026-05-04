package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedExportAsNamespaceDeclaration(t *testing.T) {
	_, err := parseProgramError(`export as namespace Jayess;`)
	requireExportAsNamespaceError(t, err)
}

func TestParserRejectsUnsupportedExportAsNamespaceInsideModule(t *testing.T) {
	_, err := parseProgramError(`
		export const value = 1;
		export as namespace Jayess;
	`)
	requireExportAsNamespaceError(t, err)
}

func requireExportAsNamespaceError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected export as namespace declaration error")
	}
	if !strings.Contains(err.Error(), "export as namespace declarations are not supported") {
		t.Fatalf("expected unsupported export as namespace diagnostic, got %v", err)
	}
}
