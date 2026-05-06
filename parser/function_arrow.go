package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) isArrowFunctionStart() bool {
	return p.isSingleParamArrowStart() || p.isParenthesizedArrowStart()
}

func (p *Parser) isSingleParamArrowStart() bool {
	if p.current.Type != lexer.TokenIdent {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return p.current.Type == lexer.TokenArrow && !p.hasLineTerminatorBeforeCurrent()
}

func (p *Parser) isParenthesizedArrowStart() bool {
	if p.current.Type != lexer.TokenLParen {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	if _, err := p.parseParameterList(); err != nil {
		return false
	}
	return p.current.Type == lexer.TokenArrow && !p.hasLineTerminatorBeforeCurrent()
}

func (p *Parser) parseArrowFunction() (ast.Expression, error) {
	start := p.current
	var (
		params []ast.Parameter
		err    error
	)
	if p.current.Type == lexer.TokenIdent {
		name := p.current.Literal
		params = []ast.Parameter{{
			Name:    name,
			Pattern: &ast.BindingName{Name: name},
		}}
		p.advance()
	} else {
		params, err = p.parseParameterList()
		if err != nil {
			return nil, err
		}
	}
	if err := p.expect(lexer.TokenArrow); err != nil {
		return nil, err
	}
	return p.parseArrowBody(baseFrom(start), params)
}

func (p *Parser) parseArrowBody(start ast.BaseNode, params []ast.Parameter) (ast.Expression, error) {
	fn := &ast.FunctionExpression{BaseNode: start, Params: params, IsArrowFunction: true}
	if p.current.Type == lexer.TokenLBrace {
		body, err := p.parseBlockStatements()
		if err != nil {
			return nil, err
		}
		fn.Body = body
		return fn, nil
	}
	body, err := p.parseConditional()
	if err != nil {
		return nil, err
	}
	fn.ExpressionBody = body
	return fn, nil
}
