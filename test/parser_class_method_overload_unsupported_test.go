package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedClassMethodOverloadDeclaration(t *testing.T) {
	_, err := parseProgramError(`class Widget { render(value); }`)
	requireClassMethodOverloadError(t, err)
}

func TestParserRejectsUnsupportedConstructorOverloadDeclaration(t *testing.T) {
	_, err := parseProgramError(`class Widget { constructor(value); }`)
	requireClassMethodOverloadError(t, err)
}

func TestParserRejectsUnsupportedStaticClassMethodOverloadDeclaration(t *testing.T) {
	_, err := parseProgramError(`class Widget { static create(value); }`)
	requireClassMethodOverloadError(t, err)
}

func requireClassMethodOverloadError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected class method overload declaration error")
	}
	if !strings.Contains(err.Error(), "class method overload declarations are not supported") {
		t.Fatalf("expected unsupported class method overload diagnostic, got %v", err)
	}
}
