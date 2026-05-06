package parser

import "jayess-go/lexer"

func (p *Parser) isUnsupportedAsyncGenericArrowTypeParametersStart() bool {
	if p.current.Type != lexer.TokenAsync {
		return false
	}
	start := p.current
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Line > start.Line {
		return false
	}
	return p.isUnsupportedGenericArrowTypeParametersStart()
}

func (p *Parser) isUnsupportedGenericArrowTypeParametersStart() bool {
	if p.current.Type != lexer.TokenLt {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	if !p.skipGenericTypeParameterListForArrow() {
		return false
	}
	if p.current.Type != lexer.TokenLParen {
		return false
	}
	if _, err := p.parseParameterList(); err != nil {
		return false
	}
	return p.current.Type == lexer.TokenArrow && !p.hasLineTerminatorBeforeCurrent()
}

func (p *Parser) skipGenericTypeParameterListForArrow() bool {
	p.advance()
	if p.current.Type != lexer.TokenIdent {
		return false
	}
	depth := 1
	for i := 0; i < 64; i++ {
		switch p.current.Type {
		case lexer.TokenLt:
			depth++
		case lexer.TokenGt:
			depth--
			if depth == 0 {
				p.advance()
				return true
			}
		case lexer.TokenEOF, lexer.TokenSemicolon:
			return false
		}
		p.advance()
	}
	return false
}

func (p *Parser) isUnsupportedAsyncArrowReturnTypeAnnotationStart() bool {
	if p.current.Type != lexer.TokenAsync {
		return false
	}
	start := p.current
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Line > start.Line {
		return false
	}
	return p.isUnsupportedArrowReturnTypeAnnotationStart()
}

func (p *Parser) isUnsupportedArrowReturnTypeAnnotationStart() bool {
	if p.current.Type != lexer.TokenLParen {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	if _, err := p.parseParameterList(); err != nil {
		return false
	}
	if p.current.Type != lexer.TokenColon {
		return false
	}
	return p.hasArrowAfterUnsupportedReturnAnnotation()
}

func (p *Parser) hasArrowAfterUnsupportedReturnAnnotation() bool {
	p.advance()
	for i := 0; i < 64; i++ {
		switch p.current.Type {
		case lexer.TokenArrow:
			return !p.hasLineTerminatorBeforeCurrent()
		case lexer.TokenEOF, lexer.TokenSemicolon:
			return false
		default:
			p.advance()
		}
	}
	return false
}

func (p *Parser) unsupportedArrowReturnAnnotationError() error {
	state := p.snapshot()
	defer p.restore(state)

	if p.current.Type == lexer.TokenAsync {
		p.advance()
	}
	if _, err := p.parseParameterList(); err != nil {
		return p.unsupportedReturnTypeAnnotationError()
	}
	if p.current.Type != lexer.TokenColon {
		return p.unsupportedReturnTypeAnnotationError()
	}
	return p.unsupportedReturnAnnotationError()
}
