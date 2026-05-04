package diagnostics

import "fmt"

type SourceLocation struct {
	File   string
	Line   int
	Column int
}

func (location SourceLocation) String() string {
	if location.File == "" {
		return fmt.Sprintf("%d:%d", location.Line, location.Column)
	}
	return fmt.Sprintf("%s:%d:%d", location.File, location.Line, location.Column)
}

type Severity string

const (
	ErrorSeverity   Severity = "error"
	WarningSeverity Severity = "warning"
	InfoSeverity    Severity = "info"
)

type Diagnostic struct {
	Code     string
	Message  string
	Severity Severity
	Location SourceLocation
	Detail   string
}

func (diagnostic Diagnostic) String() string {
	if diagnostic.Location.Line == 0 {
		return fmt.Sprintf("%s: %s", diagnostic.Severity, diagnostic.Message)
	}
	return fmt.Sprintf("%s: %s at %s", diagnostic.Severity, diagnostic.Message, diagnostic.Location.String())
}
