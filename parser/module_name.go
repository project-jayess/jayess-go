package parser

import "jayess-go/lexer"

func (p *Parser) parseModuleSpecifierName(label string) (string, error) {
	if isModuleSpecifierNameToken(p.current.Type) {
		name := p.current.Literal
		p.advance()
		return name, nil
	}
	return "", p.errorAtCurrent("expected %s, got %s", label, p.current.Type)
}

func isModuleSpecifierNameToken(tokenType lexer.TokenType) bool {
	if tokenType == lexer.TokenNumber {
		return false
	}
	return isObjectPropertyNameToken(tokenType) || tokenType == lexer.TokenString
}
