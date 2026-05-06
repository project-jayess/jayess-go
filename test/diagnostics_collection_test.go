package test

import (
	"testing"

	"jayess-go/diagnostics"
	"jayess-go/lexer"
	"jayess-go/parser"
	"jayess-go/semantic"
)

func TestDiagnosticsCollectionSortsDeterministically(t *testing.T) {
	var collection diagnostics.Collection
	collection.AddError("JY-SEMANTIC", diagnostics.SourceLocation{File: "b.js", Line: 1, Column: 1}, "semantic")
	collection.AddError("JY-PARSE", diagnostics.SourceLocation{File: "a.js", Line: 2, Column: 1}, "parse later")
	collection.AddError("JY-PARSE", diagnostics.SourceLocation{File: "a.js", Line: 1, Column: 5}, "parse first")

	values := collection.Diagnostics()
	if len(values) != 3 {
		t.Fatalf("expected three diagnostics, got %#v", values)
	}
	if values[0].Message != "parse first" || values[1].Message != "parse later" || values[2].Message != "semantic" {
		t.Fatalf("expected deterministic diagnostic order, got %#v", values)
	}
	if !collection.HasErrors() {
		t.Fatal("expected collection to report errors")
	}
}

func TestDiagnosticsAdaptersCollectParserAndSemanticErrors(t *testing.T) {
	var collection diagnostics.Collection
	_, parseErr := parser.New(lexer.New("const value = ;")).ParseProgram()
	if diagnostic, ok := diagnostics.FromParserError("parse.js", parseErr); ok {
		collection.Add(diagnostic)
	}
	program := parseProgram(t, "missing;")
	analyzer := semantic.New()
	semanticErr := analyzer.Analyze(program)
	if diagnostic, ok := diagnostics.FromSemanticError("semantic.js", semanticErr); ok {
		collection.Add(diagnostic)
	}

	values := collection.Diagnostics()
	if len(values) != 2 {
		t.Fatalf("expected parser and semantic diagnostics, got %#v", values)
	}
	if values[0].Code != "JY-PARSE" || values[1].Code != "JY-SEMANTIC" {
		t.Fatalf("expected parse and semantic codes, got %#v", values)
	}
}

func TestDiagnosticsCarryNotesAndSpans(t *testing.T) {
	var collection diagnostics.Collection
	collection.Add(diagnostics.CompilerDiagnostic{
		Code:     "JY-MODULE",
		Severity: diagnostics.ErrorSeverity,
		Span: diagnostics.SourceSpan{
			Start: diagnostics.SourceLocation{File: "main.js", Line: 1, Column: 8},
			End:   diagnostics.SourceLocation{File: "main.js", Line: 1, Column: 19},
		},
		Message: "missing export",
		Notes: []diagnostics.Note{
			{Message: "imported here", Location: diagnostics.SourceLocation{File: "main.js", Line: 1, Column: 1}},
		},
	})
	values := collection.Diagnostics()
	if values[0].Span.End.Column != 19 || len(values[0].Notes) != 1 {
		t.Fatalf("expected span and note to be preserved, got %#v", values[0])
	}
}
