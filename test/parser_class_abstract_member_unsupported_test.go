package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedAbstractClassMethod(t *testing.T) {
	_, err := parseProgramError(`class Box { abstract value() { return 1; } }`)
	requireAbstractClassMemberError(t, err)
}

func TestParserRejectsUnsupportedAbstractClassField(t *testing.T) {
	_, err := parseProgramError(`class Box { abstract value = 1; }`)
	requireAbstractClassMemberError(t, err)
}

func TestParserAllowsClassFieldNamedAbstract(t *testing.T) {
	parseProgram(t, `class Box { abstract = 1; }`)
}

func requireAbstractClassMemberError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected abstract class member error")
	}
	if !strings.Contains(err.Error(), "abstract modifiers are not supported") {
		t.Fatalf("expected abstract modifier diagnostic, got %v", err)
	}
}
