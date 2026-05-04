package test

import (
	"strings"
	"testing"
)

func TestParserRejectsImportAssertionsWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`import data from "./data.json" assert { type: "json" };`)
	if err == nil {
		t.Fatalf("expected unsupported import assertion error")
	}
	if !strings.Contains(err.Error(), "import attributes are not supported") {
		t.Fatalf("expected clear import attributes diagnostic, got %v", err)
	}
}

func TestParserRejectsImportAttributesWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`import data from "./data.json" with { type: "json" };`)
	if err == nil {
		t.Fatalf("expected unsupported import attributes error")
	}
	if !strings.Contains(err.Error(), "import attributes are not supported") {
		t.Fatalf("expected clear import attributes diagnostic, got %v", err)
	}
}

func TestParserRejectsSideEffectImportAttributesWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`import "./setup.js" with { phase: "init" };`)
	if err == nil {
		t.Fatalf("expected unsupported side-effect import attributes error")
	}
	if !strings.Contains(err.Error(), "import attributes are not supported") {
		t.Fatalf("expected clear import attributes diagnostic, got %v", err)
	}
}

func TestParserRejectsReExportImportAttributesWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`export * from "./data.json" with { type: "json" };`)
	if err == nil {
		t.Fatalf("expected unsupported re-export import attributes error")
	}
	if !strings.Contains(err.Error(), "import attributes are not supported") {
		t.Fatalf("expected clear import attributes diagnostic, got %v", err)
	}
}
