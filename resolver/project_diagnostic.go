package resolver

import (
	"fmt"

	"jayess-go/ast"
	"jayess-go/parser"
)

type ProjectDiagnostic struct {
	Path    string
	Line    int
	Column  int
	Message string
}

func (d ProjectDiagnostic) Error() string {
	if d.Path == "" {
		return d.Message
	}
	if d.Line > 0 {
		return fmt.Sprintf("%s:%d:%d: %s", d.Path, d.Line, d.Column, d.Message)
	}
	return fmt.Sprintf("%s: %s", d.Path, d.Message)
}

func projectDiagnostic(path string, pos ast.SourcePos, message string) ProjectDiagnostic {
	return ProjectDiagnostic{
		Path:    path,
		Line:    pos.Line,
		Column:  pos.Column,
		Message: message,
	}
}

func parseProjectDiagnostic(path string, err error) ProjectDiagnostic {
	if diagnostic, ok := err.(*parser.DiagnosticError); ok {
		return ProjectDiagnostic{
			Path:    path,
			Line:    diagnostic.Line,
			Column:  diagnostic.Column,
			Message: diagnostic.Message,
		}
	}
	return ProjectDiagnostic{Path: path, Message: err.Error()}
}
