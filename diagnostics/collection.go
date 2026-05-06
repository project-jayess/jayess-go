package diagnostics

import "sort"

type SourceSpan struct {
	Start SourceLocation
	End   SourceLocation
}

type Note struct {
	Message  string
	Location SourceLocation
}

type CompilerDiagnostic struct {
	Code     string
	Severity Severity
	Span     SourceSpan
	Message  string
	Notes    []Note
}

type Collection struct {
	diagnostics []CompilerDiagnostic
}

func (collection *Collection) Add(diagnostic CompilerDiagnostic) {
	collection.diagnostics = append(collection.diagnostics, diagnostic)
}

func (collection *Collection) AddError(code string, location SourceLocation, message string) {
	collection.Add(CompilerDiagnostic{
		Code:     code,
		Severity: ErrorSeverity,
		Span:     SourceSpan{Start: location, End: location},
		Message:  message,
	})
}

func (collection *Collection) Diagnostics() []CompilerDiagnostic {
	diagnostics := append([]CompilerDiagnostic(nil), collection.diagnostics...)
	sort.SliceStable(diagnostics, func(i, j int) bool {
		return compareDiagnostics(diagnostics[i], diagnostics[j]) < 0
	})
	return diagnostics
}

func (collection *Collection) HasErrors() bool {
	for _, diagnostic := range collection.diagnostics {
		if diagnostic.Severity == ErrorSeverity {
			return true
		}
	}
	return false
}

func compareDiagnostics(left CompilerDiagnostic, right CompilerDiagnostic) int {
	if left.Span.Start.File != right.Span.Start.File {
		if left.Span.Start.File < right.Span.Start.File {
			return -1
		}
		return 1
	}
	if left.Span.Start.Line != right.Span.Start.Line {
		return left.Span.Start.Line - right.Span.Start.Line
	}
	if left.Span.Start.Column != right.Span.Start.Column {
		return left.Span.Start.Column - right.Span.Start.Column
	}
	if left.Code < right.Code {
		return -1
	}
	if left.Code > right.Code {
		return 1
	}
	return 0
}
