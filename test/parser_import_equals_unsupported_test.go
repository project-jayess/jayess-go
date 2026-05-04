package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedImportEqualsDeclaration(t *testing.T) {
	_, err := parseProgramError(`import fs = require("fs");`)
	requireImportEqualsError(t, err)
}

func TestParserRejectsUnsupportedQualifiedImportEqualsDeclaration(t *testing.T) {
	_, err := parseProgramError(`import Path = Node.Path;`)
	requireImportEqualsError(t, err)
}

func requireImportEqualsError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected import equals declaration error")
	}
	if !strings.Contains(err.Error(), "import equals declarations are not supported") {
		t.Fatalf("expected unsupported import equals diagnostic, got %v", err)
	}
}
