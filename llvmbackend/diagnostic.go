package llvmbackend

import (
	"errors"
	"fmt"

	"jayess-go/ast"
)

type DiagnosticError struct {
	Line   int
	Column int
	Err    error
}

func (err DiagnosticError) Error() string {
	return fmt.Sprintf("%d:%d: %s", err.Line, err.Column, err.Err)
}

func (err DiagnosticError) Unwrap() error {
	return err.Err
}

func diagnosticError(node any, err error) error {
	if err == nil {
		return nil
	}
	var existing DiagnosticError
	if errors.As(err, &existing) {
		return err
	}
	position := ast.PositionOf(node)
	if position.Line == 0 || position.Column == 0 {
		return err
	}
	return DiagnosticError{Line: position.Line, Column: position.Column, Err: err}
}
