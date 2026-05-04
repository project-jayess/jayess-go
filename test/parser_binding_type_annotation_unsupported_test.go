package test

import (
	"strings"
	"testing"
)

func TestParserRejectsArrayBindingTypeAnnotationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const [value: number] = values;`)
	assertTypeAnnotationDiagnostic(t, err)
}

func TestParserRejectsNestedObjectBindingTypeAnnotationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const { value: local: string } = item;`)
	assertTypeAnnotationDiagnostic(t, err)
}

func TestParserRejectsRestArrayBindingTypeAnnotationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const [...rest: string[]] = values;`)
	assertTypeAnnotationDiagnostic(t, err)
}

func TestParserRejectsRestObjectBindingTypeAnnotationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const { ...rest: Rest } = item;`)
	assertTypeAnnotationDiagnostic(t, err)
}

func assertTypeAnnotationDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected unsupported type annotation error")
	}
	if !strings.Contains(err.Error(), "type annotations are not supported") {
		t.Fatalf("expected clear type annotation diagnostic, got %v", err)
	}
}
