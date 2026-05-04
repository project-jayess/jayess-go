package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) isImportMetaStart() bool {
	if p.current.Type != lexer.TokenImport {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if !p.match(lexer.TokenDot) {
		return false
	}
	return p.current.Type == lexer.TokenIdent && p.current.Literal == "meta"
}

func (p *Parser) isDynamicImportStart() bool {
	if p.current.Type != lexer.TokenImport {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return p.current.Type == lexer.TokenLParen
}

func (p *Parser) parseImportMetaExpression() (ast.Expression, error) {
	start := p.current
	p.advance()
	if err := p.expect(lexer.TokenDot); err != nil {
		return nil, err
	}
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "meta" {
		return nil, p.errorAtCurrent("expected import.meta")
	}
	p.advance()
	return &ast.ImportMetaExpression{BaseNode: baseFrom(start)}, nil
}
