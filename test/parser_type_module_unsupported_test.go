package test

import (
	"strings"
	"testing"
)

func TestParserRejectsTypeOnlyImportWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`import type { User } from "./types.js";`)
	if err == nil {
		t.Fatalf("expected unsupported type-only import error")
	}
	if !strings.Contains(err.Error(), "type-only imports and exports are not supported") {
		t.Fatalf("expected clear type-only module diagnostic, got %v", err)
	}
}

func TestParserRejectsTypeOnlyExportListWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`export type { User };`)
	if err == nil {
		t.Fatalf("expected unsupported type-only export list error")
	}
	if !strings.Contains(err.Error(), "type-only imports and exports are not supported") {
		t.Fatalf("expected clear type-only module diagnostic, got %v", err)
	}
}

func TestParserRejectsTypeOnlyImportSpecifierWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`import { type User } from "./types.js";`)
	if err == nil {
		t.Fatalf("expected unsupported type-only import specifier error")
	}
	if !strings.Contains(err.Error(), "type-only imports and exports are not supported") {
		t.Fatalf("expected clear type-only module diagnostic, got %v", err)
	}
}

func TestParserRejectsTypeOnlyExportSpecifierWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`export { type User };`)
	if err == nil {
		t.Fatalf("expected unsupported type-only export specifier error")
	}
	if !strings.Contains(err.Error(), "type-only imports and exports are not supported") {
		t.Fatalf("expected clear type-only module diagnostic, got %v", err)
	}
}

func TestParserAllowsValueSpecifierNamedType(t *testing.T) {
	parseProgram(t, `
		import { type as valueType } from "./types.js";
		export { type as valueType };
	`)
}

func TestParserRejectsExportedTypeAliasWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`export type User = { name: string };`)
	if err == nil {
		t.Fatalf("expected unsupported exported type alias error")
	}
	if !strings.Contains(err.Error(), "type-only imports and exports are not supported") {
		t.Fatalf("expected clear type-only module diagnostic, got %v", err)
	}
}
