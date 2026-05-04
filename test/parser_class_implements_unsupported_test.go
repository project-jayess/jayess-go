package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedClassImplementsClause(t *testing.T) {
	_, err := parseProgramError(`class Widget implements View {}`)
	if err == nil {
		t.Fatalf("expected implements clause error")
	}
	if !strings.Contains(err.Error(), "implements clauses are not supported") {
		t.Fatalf("expected unsupported implements diagnostic, got %v", err)
	}
}

func TestParserRejectsUnsupportedClassExtendsImplementsClause(t *testing.T) {
	_, err := parseProgramError(`class Widget extends Base implements View {}`)
	if err == nil {
		t.Fatalf("expected implements clause error")
	}
	if !strings.Contains(err.Error(), "implements clauses are not supported") {
		t.Fatalf("expected unsupported implements diagnostic, got %v", err)
	}
}
