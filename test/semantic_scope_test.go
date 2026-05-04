package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/parser"
	"jayess-go/semantic"
)

func TestSemanticAllowsBlockScopeAndShadowing(t *testing.T) {
	err := analyzeSource(t, `
		var value = 1;
		{
			var value = 2;
			value;
		}
		value;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUseBeforeDeclaration(t *testing.T) {
	err := analyzeSource(t, `
		value;
		var value = 1;
	`)
	requireSemanticError(t, err, "use of value before declaration")
}

func TestSemanticRejectsBlockLocalOutsideBlock(t *testing.T) {
	err := analyzeSource(t, `
		{
			var inner = 1;
		}
		inner;
	`)
	requireSemanticError(t, err, "use of inner before declaration")
}

func TestSemanticRejectsDuplicateDeclarationInSameScope(t *testing.T) {
	err := analyzeSource(t, `
		var value = 1;
		const value = 2;
	`)
	requireSemanticError(t, err, "duplicate declaration value")
}

func TestSemanticAllowsFunctionParamsAndLocals(t *testing.T) {
	err := analyzeSource(t, `
		function main(arg) {
			const value = arg;
			return value;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsFunctionCallsAfterDeclaration(t *testing.T) {
	err := analyzeSource(t, `
		function make(x) {
			return x;
		}
		const value = make(1);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsFunctionCallsBeforeDeclaration(t *testing.T) {
	err := analyzeSource(t, `
		const value = make(1);
		function make(x) {
			return x;
		}
	`)
	requireSemanticError(t, err, "use of make before declaration")
}

func analyzeSource(t *testing.T, source string) error {
	t.Helper()
	program, err := parser.New(lexer.New(source)).ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	return semantic.New().Analyze(program)
}

func requireSemanticError(t *testing.T, err error, message string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected semantic error containing %q", message)
	}
	if !strings.Contains(err.Error(), message) {
		t.Fatalf("expected semantic error containing %q, got %q", message, err.Error())
	}
}
