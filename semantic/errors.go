package semantic

import (
	"fmt"

	"jayess-go/ast"
)

type DiagnosticError struct {
	Line    int
	Column  int
	Message string
}

func (e *DiagnosticError) Error() string {
	if e == nil {
		return ""
	}
	if e.Line > 0 {
		return fmt.Sprintf("%d:%d: %s", e.Line, e.Column, e.Message)
	}
	return e.Message
}

func errorAt(node any, format string, args ...any) error {
	pos := ast.PositionOf(node)
	return &DiagnosticError{
		Line:    pos.Line,
		Column:  pos.Column,
		Message: fmt.Sprintf(format, args...),
	}
}

func errorf(format string, args ...any) error {
	return &DiagnosticError{Message: fmt.Sprintf(format, args...)}
}
