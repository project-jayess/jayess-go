package parser

import (
	"fmt"

	"jayess-go/ast"
	"jayess-go/lexer"
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

func (p *Parser) errorAtCurrent(format string, args ...any) error {
	return errorAtToken(p.current, format, args...)
}

func errorAtToken(token lexer.Token, format string, args ...any) error {
	if token.Type == lexer.TokenIllegal {
		return &DiagnosticError{
			Line:    token.Line,
			Column:  token.Column,
			Message: token.Literal,
		}
	}
	return &DiagnosticError{
		Line:    token.Line,
		Column:  token.Column,
		Message: fmt.Sprintf(format, args...),
	}
}

func errorAtPosition(pos ast.SourcePos, format string, args ...any) error {
	return &DiagnosticError{
		Line:    pos.Line,
		Column:  pos.Column,
		Message: fmt.Sprintf(format, args...),
	}
}
