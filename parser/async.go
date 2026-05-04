package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) isAsyncFunctionStart() bool {
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
	return p.current.Type == lexer.TokenFunction
}

func (p *Parser) isUnsupportedAsyncLineTerminatorStart() bool {
	if p.current.Type != lexer.TokenAsync {
		return false
	}
	start := p.current
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Line <= start.Line {
		return false
	}
	if p.current.Type == lexer.TokenFunction {
		return true
	}
	return p.isSingleParamArrowStart() || p.isParenthesizedArrowStart()
}

func (p *Parser) parseAsyncFunctionDeclaration() (ast.Statement, error) {
	start := p.current
	p.advance()
	return p.parseFunctionDeclarationWithAsync(start, true)
}

func (p *Parser) parseAsyncFunctionExpression() (ast.Expression, error) {
	start := p.current
	p.advance()
	return p.parseFunctionExpressionWithAsync(start, true)
}

func (p *Parser) isAnonymousAsyncFunctionExpressionStart() bool {
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
	if p.current.Type != lexer.TokenFunction {
		return false
	}
	p.advance()
	if p.current.Type == lexer.TokenStar {
		p.advance()
	}
	return p.current.Type == lexer.TokenLParen
}

func (p *Parser) isAsyncArrowFunctionStart() bool {
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
	return p.isSingleParamArrowStart() || p.isParenthesizedArrowStart()
}

func (p *Parser) parseAsyncArrowFunction() (ast.Expression, error) {
	start := p.current
	p.advance()
	expr, err := p.parseArrowFunction()
	if err != nil {
		return nil, err
	}
	fn := expr.(*ast.FunctionExpression)
	fn.BaseNode = baseFrom(start)
	fn.IsAsync = true
	return fn, nil
}

func (p *Parser) parseAwaitExpression(token lexer.Token) (ast.Expression, error) {
	p.advance()
	value, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	return &ast.AwaitExpression{BaseNode: baseFrom(token), Value: value}, nil
}
