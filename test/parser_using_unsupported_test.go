package test

import (
	"strings"
	"testing"
)

func TestParserRejectsUnsupportedUsingDeclaration(t *testing.T) {
	_, err := parseProgramError(`using handle = open();`)
	if err == nil {
		t.Fatalf("expected using declaration error")
	}
	if !strings.Contains(err.Error(), "using declarations are not supported") {
		t.Fatalf("expected unsupported using diagnostic, got %v", err)
	}
}

func TestParserRejectsUnsupportedUsingDestructuringDeclaration(t *testing.T) {
	_, err := parseProgramError(`using { handle } = open();`)
	if err == nil {
		t.Fatalf("expected using destructuring declaration error")
	}
	if !strings.Contains(err.Error(), "using declarations are not supported") {
		t.Fatalf("expected unsupported using diagnostic, got %v", err)
	}
}

func TestParserRejectsUnsupportedAwaitUsingDeclaration(t *testing.T) {
	_, err := parseProgramError(`async function run() { await using handle = open(); }`)
	if err == nil {
		t.Fatalf("expected await using declaration error")
	}
	if !strings.Contains(err.Error(), "using declarations are not supported") {
		t.Fatalf("expected unsupported using diagnostic, got %v", err)
	}
}
