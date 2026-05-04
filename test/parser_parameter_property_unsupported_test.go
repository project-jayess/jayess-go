package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedConstructorPublicParameterProperty(t *testing.T) {
	_, err := parseProgramError(`class Widget { constructor(public value) {} }`)
	requireParameterPropertyError(t, err)
}

func TestParserRejectsUnsupportedConstructorPrivateParameterProperty(t *testing.T) {
	_, err := parseProgramError(`class Widget { constructor(private value) {} }`)
	requireParameterPropertyError(t, err)
}

func TestParserRejectsUnsupportedConstructorReadonlyParameterProperty(t *testing.T) {
	_, err := parseProgramError(`class Widget { constructor(readonly value) {} }`)
	requireParameterPropertyError(t, err)
}

func TestParserStillAllowsReadonlyParameterName(t *testing.T) {
	program := parseProgram(t, `function read(readonly) { return readonly; }`)
	if len(program.Statements) != 1 {
		t.Fatalf("expected one statement, got %d", len(program.Statements))
	}
}

func requireParameterPropertyError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected parameter property modifier error")
	}
	if !strings.Contains(err.Error(), "parameter property modifiers are not supported") {
		t.Fatalf("expected unsupported parameter property diagnostic, got %v", err)
	}
}
