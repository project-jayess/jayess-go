package test

import (
	"strings"
	"testing"
)

func TestParserRejectsArrowReturnTypeAnnotationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const fn = (value): number => value;`)
	requireArrowReturnTypeError(t, err)
}

func TestParserRejectsAsyncArrowReturnTypeAnnotationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const fn = async (value): number => value;`)
	requireArrowReturnTypeError(t, err)
}

func TestParserRejectsArrowTypePredicateReturnWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const fn = (value): value is Widget => true;`)
	requireArrowTypePredicateError(t, err)
}

func requireArrowReturnTypeError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected unsupported arrow return type annotation error")
	}
	if !strings.Contains(err.Error(), "return type annotations are not supported") {
		t.Fatalf("expected clear return type annotation diagnostic, got %v", err)
	}
}

func requireArrowTypePredicateError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected unsupported arrow type predicate return error")
	}
	if !strings.Contains(err.Error(), "type predicate and assertion return annotations are not supported") {
		t.Fatalf("expected clear type predicate return diagnostic, got %v", err)
	}
}
