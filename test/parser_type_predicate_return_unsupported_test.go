package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedTypePredicateReturn(t *testing.T) {
	_, err := parseProgramError(`function isWidget(value): value is Widget { return true; }`)
	requireTypePredicateReturnError(t, err)
}

func TestParserRejectsUnsupportedAssertionReturn(t *testing.T) {
	_, err := parseProgramError(`function assertWidget(value): asserts value { return; }`)
	requireTypePredicateReturnError(t, err)
}

func TestParserRejectsUnsupportedClassMethodTypePredicateReturn(t *testing.T) {
	_, err := parseProgramError(`class Box { isWidget(value): value is Widget { return true; } }`)
	requireTypePredicateReturnError(t, err)
}

func TestParserRejectsUnsupportedObjectMethodTypePredicateReturn(t *testing.T) {
	_, err := parseProgramError(`const box = { isWidget(value): value is Widget { return true; } };`)
	requireTypePredicateReturnError(t, err)
}

func TestParserStillRejectsPlainReturnTypesAsReturnTypeAnnotations(t *testing.T) {
	_, err := parseProgramError(`function count(): number { return 1; }`)
	if err == nil {
		t.Fatalf("expected return type annotation error")
	}
	if !strings.Contains(err.Error(), "return type annotations are not supported") {
		t.Fatalf("expected return type annotation diagnostic, got %v", err)
	}
}

func requireTypePredicateReturnError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected type predicate return error")
	}
	if !strings.Contains(err.Error(), "type predicate and assertion return annotations are not supported") {
		t.Fatalf("expected type predicate return diagnostic, got %v", err)
	}
}
