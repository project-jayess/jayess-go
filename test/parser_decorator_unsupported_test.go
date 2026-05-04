package test

import (
	"strings"
	"testing"
)

func TestParserRejectsTopLevelDecoratorWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`@sealed class Widget {}`)
	if err == nil {
		t.Fatalf("expected unsupported decorator error")
	}
	if !strings.Contains(err.Error(), "decorators are not supported") {
		t.Fatalf("expected clear decorator diagnostic, got %v", err)
	}
}

func TestParserRejectsClassMemberDecoratorWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`class Widget { @readonly value = 1; }`)
	if err == nil {
		t.Fatalf("expected unsupported class member decorator error")
	}
	if !strings.Contains(err.Error(), "decorators are not supported") {
		t.Fatalf("expected clear decorator diagnostic, got %v", err)
	}
}

func TestParserRejectsParameterDecoratorWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`function use(@inject value) { return value; }`)
	if err == nil {
		t.Fatalf("expected unsupported parameter decorator error")
	}
	if !strings.Contains(err.Error(), "decorators are not supported") {
		t.Fatalf("expected clear decorator diagnostic, got %v", err)
	}
}

func TestParserRejectsClassMethodParameterDecoratorWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`class Widget { use(@inject value) { return value; } }`)
	if err == nil {
		t.Fatalf("expected unsupported class method parameter decorator error")
	}
	if !strings.Contains(err.Error(), "decorators are not supported") {
		t.Fatalf("expected clear decorator diagnostic, got %v", err)
	}
}
