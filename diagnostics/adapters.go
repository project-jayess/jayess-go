package diagnostics

import (
	"jayess-go/parser"
	"jayess-go/semantic"
)

func FromParserError(file string, err error) (CompilerDiagnostic, bool) {
	diagnostic, ok := err.(*parser.DiagnosticError)
	if !ok {
		return CompilerDiagnostic{}, false
	}
	location := SourceLocation{File: file, Line: diagnostic.Line, Column: diagnostic.Column}
	return CompilerDiagnostic{
		Code:     "JY-PARSE",
		Severity: ErrorSeverity,
		Span:     SourceSpan{Start: location, End: location},
		Message:  diagnostic.Message,
	}, true
}

func FromSemanticError(file string, err error) (CompilerDiagnostic, bool) {
	diagnostic, ok := err.(*semantic.DiagnosticError)
	if !ok {
		return CompilerDiagnostic{}, false
	}
	location := SourceLocation{File: file, Line: diagnostic.Line, Column: diagnostic.Column}
	return CompilerDiagnostic{
		Code:     "JY-SEMANTIC",
		Severity: ErrorSeverity,
		Span:     SourceSpan{Start: location, End: location},
		Message:  diagnostic.Message,
	}, true
}
