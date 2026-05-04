package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseNewExpression() (ast.Expression, error) {
	start := p.current
	p.advance()

	callee, err := p.parsePostfix()
	if err != nil {
		return nil, err
	}
	if hasOptionalChain(callee) {
		return nil, errorAtPosition(ast.PositionOf(callee), "new target cannot contain optional chaining")
	}
	callee, args := splitConstructorCall(callee)
	return &ast.NewExpression{
		BaseNode:  baseFrom(start),
		Callee:    callee,
		Arguments: args,
	}, nil
}

func (p *Parser) isNewTargetStart() bool {
	if p.current.Type != lexer.TokenNew {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if !p.match(lexer.TokenDot) {
		return false
	}
	return p.current.Type == lexer.TokenIdent && p.current.Literal == "target"
}

func (p *Parser) parseNewTargetExpression() (ast.Expression, error) {
	start := p.current
	p.advance()
	if err := p.expect(lexer.TokenDot); err != nil {
		return nil, err
	}
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "target" {
		return nil, p.errorAtCurrent("expected new.target")
	}
	p.advance()
	return &ast.NewTargetExpression{BaseNode: baseFrom(start)}, nil
}

func splitConstructorCall(expr ast.Expression) (ast.Expression, []ast.Expression) {
	switch expr := expr.(type) {
	case *ast.CallExpression:
		return &ast.Identifier{BaseNode: expr.BaseNode, Name: expr.Callee}, expr.Arguments
	case *ast.InvokeExpression:
		if !expr.Optional {
			return expr.Callee, expr.Arguments
		}
	}
	return expr, nil
}

func isNewExpressionStart(token lexer.Token) bool {
	return token.Type == lexer.TokenNew
}
