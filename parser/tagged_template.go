package parser

import (
	"jayess-go/ast"
)

func (p *Parser) parseTaggedTemplate(callee ast.Expression) (ast.Expression, error) {
	token := p.current
	p.advance()
	template, err := p.parseTemplateLiteral(token)
	if err != nil {
		return nil, err
	}
	return callExpression(callee, []ast.Expression{template}, false), nil
}
