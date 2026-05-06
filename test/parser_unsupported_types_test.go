package test

import (
	"strings"
	"testing"
)

func TestParserRejectsVariableTypeAnnotationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const value: number = 1;`)
	if err == nil {
		t.Fatalf("expected unsupported variable type annotation error")
	}
	if !strings.Contains(err.Error(), "type annotations are not supported") {
		t.Fatalf("expected clear type annotation diagnostic, got %v", err)
	}
}

func TestParserRejectsParameterTypeAnnotationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`function main(value: number) { return value; }`)
	if err == nil {
		t.Fatalf("expected unsupported parameter type annotation error")
	}
	if !strings.Contains(err.Error(), "type annotations are not supported") {
		t.Fatalf("expected clear type annotation diagnostic, got %v", err)
	}
}

func TestParserRejectsFunctionReturnTypeAnnotationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`function main(): number { return 1; }`)
	if err == nil {
		t.Fatalf("expected unsupported function return type annotation error")
	}
	if !strings.Contains(err.Error(), "return type annotations are not supported") {
		t.Fatalf("expected clear return type annotation diagnostic, got %v", err)
	}
}

func TestParserRejectsFunctionExpressionReturnTypeAnnotationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const fn = function(): number { return 1; };`)
	if err == nil {
		t.Fatalf("expected unsupported function expression return type annotation error")
	}
	if !strings.Contains(err.Error(), "return type annotations are not supported") {
		t.Fatalf("expected clear return type annotation diagnostic, got %v", err)
	}
}

func TestParserRejectsClassTypeAnnotationsWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`class Box { value: number = 1; size(): number { return this.value; } }`)
	if err == nil {
		t.Fatalf("expected unsupported class type annotation error")
	}
	if !strings.Contains(err.Error(), "type annotations are not supported") {
		t.Fatalf("expected clear class type annotation diagnostic, got %v", err)
	}
}

func TestParserRejectsObjectMethodReturnTypeAnnotationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const item = { size(): number { return 1; } };`)
	if err == nil {
		t.Fatalf("expected unsupported object method return type annotation error")
	}
	if !strings.Contains(err.Error(), "return type annotations are not supported") {
		t.Fatalf("expected clear return type annotation diagnostic, got %v", err)
	}
}

func TestParserRejectsFunctionGenericTypeParametersWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`function identity<T>(value) { return value; }`)
	if err == nil {
		t.Fatalf("expected unsupported function generic type parameter error")
	}
	if !strings.Contains(err.Error(), "generic type parameters are not supported") {
		t.Fatalf("expected clear generic type parameter diagnostic, got %v", err)
	}
}

func TestParserRejectsFunctionExpressionGenericTypeParametersWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const identity = function id<T>(value) { return value; };`)
	if err == nil {
		t.Fatalf("expected unsupported function expression generic type parameter error")
	}
	if !strings.Contains(err.Error(), "generic type parameters are not supported") {
		t.Fatalf("expected clear generic type parameter diagnostic, got %v", err)
	}
}

func TestParserRejectsClassGenericTypeParametersWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`class Box<T> { constructor(value) { this.value = value; } }`)
	if err == nil {
		t.Fatalf("expected unsupported class generic type parameter error")
	}
	if !strings.Contains(err.Error(), "generic type parameters are not supported") {
		t.Fatalf("expected clear generic type parameter diagnostic, got %v", err)
	}
}

func TestParserRejectsMethodGenericTypeParametersWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`class Box { value<T>() { return 1; } }`)
	if err == nil {
		t.Fatalf("expected unsupported class method generic type parameter error")
	}
	if !strings.Contains(err.Error(), "generic type parameters are not supported") {
		t.Fatalf("expected clear generic type parameter diagnostic, got %v", err)
	}
}

func TestParserRejectsObjectMethodGenericTypeParametersWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`const item = { value<T>() { return 1; } };`)
	if err == nil {
		t.Fatalf("expected unsupported object method generic type parameter error")
	}
	if !strings.Contains(err.Error(), "generic type parameters are not supported") {
		t.Fatalf("expected clear generic type parameter diagnostic, got %v", err)
	}
}
