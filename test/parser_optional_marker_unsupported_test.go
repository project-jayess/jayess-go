package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedOptionalParameter(t *testing.T) {
	_, err := parseProgramError(`function read(value?) { return value; }`)
	requireOptionalParameterError(t, err)
}

func TestParserRejectsUnsupportedOptionalObjectProperty(t *testing.T) {
	_, err := parseProgramError(`const item = { value?: 1 };`)
	requireOptionalPropertyError(t, err)
}

func TestParserRejectsUnsupportedOptionalObjectMethod(t *testing.T) {
	_, err := parseProgramError(`const item = { value?() { return 1; } };`)
	requireOptionalPropertyError(t, err)
}

func TestParserRejectsUnsupportedOptionalClassField(t *testing.T) {
	_, err := parseProgramError(`class Box { value? = 1; }`)
	requireOptionalPropertyError(t, err)
}

func TestParserRejectsUnsupportedOptionalClassMethod(t *testing.T) {
	_, err := parseProgramError(`class Box { value?() { return 1; } }`)
	requireOptionalPropertyError(t, err)
}

func TestParserRejectsUnsupportedOptionalVariableBinding(t *testing.T) {
	_, err := parseProgramError(`var value? = 1;`)
	requireOptionalBindingError(t, err)
}

func TestParserRejectsUnsupportedOptionalArrayBinding(t *testing.T) {
	_, err := parseProgramError(`const [value?] = values;`)
	requireOptionalBindingError(t, err)
}

func TestParserRejectsUnsupportedOptionalObjectBinding(t *testing.T) {
	_, err := parseProgramError(`const { value? } = item;`)
	requireOptionalBindingError(t, err)
}

func TestParserRejectsUnsupportedOptionalNestedObjectBinding(t *testing.T) {
	_, err := parseProgramError(`const { value: local? } = item;`)
	requireOptionalBindingError(t, err)
}

func TestParserRejectsUnsupportedOptionalRestArrayBinding(t *testing.T) {
	_, err := parseProgramError(`const [...rest?] = values;`)
	requireOptionalBindingError(t, err)
}

func TestParserRejectsUnsupportedOptionalRestObjectBinding(t *testing.T) {
	_, err := parseProgramError(`const { ...rest? } = item;`)
	requireOptionalBindingError(t, err)
}

func requireOptionalParameterError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected optional parameter error")
	}
	if !strings.Contains(err.Error(), "optional parameters are not supported") {
		t.Fatalf("expected optional parameter diagnostic, got %v", err)
	}
}

func requireOptionalPropertyError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected optional property error")
	}
	if !strings.Contains(err.Error(), "optional properties and methods are not supported") {
		t.Fatalf("expected optional property diagnostic, got %v", err)
	}
}

func requireOptionalBindingError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected optional binding error")
	}
	if !strings.Contains(err.Error(), "optional bindings are not supported") {
		t.Fatalf("expected optional binding diagnostic, got %v", err)
	}
}
