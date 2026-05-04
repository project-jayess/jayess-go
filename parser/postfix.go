package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parsePostfix() (ast.Expression, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		switch p.current.Type {
		case lexer.TokenTemplate:
			expr, err = p.parseTaggedTemplate(expr)
			if err != nil {
				return nil, err
			}
		case lexer.TokenLParen:
			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}
			expr = callExpression(expr, args, false)
		case lexer.TokenLBracket:
			expr, err = p.parseIndexExpression(expr, false)
			if err != nil {
				return nil, err
			}
		case lexer.TokenDot:
			expr, err = p.parseMemberExpression(expr, false)
			if err != nil {
				return nil, err
			}
		case lexer.TokenQuestionDot:
			expr, err = p.parseOptionalPostfix(expr)
			if err != nil {
				return nil, err
			}
		case lexer.TokenIncrement, lexer.TokenDecrement:
			if p.hasLineTerminatorBeforeCurrent() {
				return expr, nil
			}
			return p.parsePostfixUpdate(expr)
		default:
			if p.current.Type == lexer.TokenArrow && p.hasLineTerminatorBeforeCurrent() {
				return nil, p.unsupportedArrowLineTerminatorError()
			}
			if !p.hasLineTerminatorBeforeCurrent() {
				if err := p.unsupportedExpressionSuffixError(); err != nil {
					return nil, err
				}
			}
			return expr, nil
		}
	}
}

func (p *Parser) hasLineTerminatorBeforeCurrent() bool {
	return p.previous.Line > 0 && p.current.Line > p.previous.Line
}

func callExpression(callee ast.Expression, args []ast.Expression, optional bool) ast.Expression {
	base := baseOf(callee)
	if ident, ok := callee.(*ast.Identifier); ok && !optional {
		return &ast.CallExpression{BaseNode: base, Callee: ident.Name, Arguments: args}
	}
	return &ast.InvokeExpression{BaseNode: base, Callee: callee, Arguments: args, Optional: optional}
}
