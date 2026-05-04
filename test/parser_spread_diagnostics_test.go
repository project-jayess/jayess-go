package test

import (
	"strings"
	"testing"
)

func TestParserRejectsMissingArraySpreadExpression(t *testing.T) {
	_, err := parseProgramError(`const values = [...];`)
	if err == nil {
		t.Fatalf("expected missing array spread expression error")
	}
	if !strings.Contains(err.Error(), "spread element requires an expression") {
		t.Fatalf("expected clear spread diagnostic, got %v", err)
	}
}

func TestParserRejectsMissingObjectSpreadExpression(t *testing.T) {
	_, err := parseProgramError(`const value = { ... };`)
	if err == nil {
		t.Fatalf("expected missing object spread expression error")
	}
	if !strings.Contains(err.Error(), "spread element requires an expression") {
		t.Fatalf("expected clear spread diagnostic, got %v", err)
	}
}

func TestParserRejectsMissingCallSpreadExpression(t *testing.T) {
	_, err := parseProgramError(`call(...);`)
	if err == nil {
		t.Fatalf("expected missing call spread expression error")
	}
	if !strings.Contains(err.Error(), "spread element requires an expression") {
		t.Fatalf("expected clear spread diagnostic, got %v", err)
	}
}
