package parser

import "jayess-go/lexer"

func (p *Parser) consumeStatementTerminator() error {
	if p.match(lexer.TokenSemicolon) {
		return nil
	}
	if p.current.Type == lexer.TokenEOF {
		return nil
	}
	if p.current.Type == lexer.TokenRBrace {
		return nil
	}
	if p.previous.Line > 0 && p.current.Line > p.previous.Line {
		return nil
	}
	if err := p.unsupportedExpressionSuffixError(); err != nil {
		return err
	}
	return p.errorAtCurrent("expected statement terminator, got %s", p.current.Type)
}

func (p *Parser) unsupportedExpressionSuffixError() error {
	if p.current.Type == lexer.TokenIdent {
		switch p.current.Literal {
		case "as":
			if p.isUnsupportedConstAssertionSuffix() {
				return p.unsupportedConstAssertionError()
			}
			return p.unsupportedTypeAssertionError()
		case "satisfies":
			return p.unsupportedSatisfiesExpressionError()
		}
	}
	if p.current.Type == lexer.TokenBang {
		return p.unsupportedNonNullAssertionError()
	}
	return nil
}
