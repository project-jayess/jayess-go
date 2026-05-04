package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseVariableDeclaration() (ast.Statement, error) {
	return p.parseVariableDeclarationTerminated(true)
}

func (p *Parser) parseVariableDeclarationTerminated(consumeTerminator bool) (ast.Statement, error) {
	start := p.current
	kind := declarationKind(start.Type)
	p.advance()

	pattern, name, err := p.parseBindingTarget()
	if err != nil {
		return nil, err
	}
	if p.current.Type == lexer.TokenBang {
		return nil, p.unsupportedDefiniteAssignmentAssertionError()
	}
	if p.current.Type == lexer.TokenQuestion {
		return nil, p.unsupportedOptionalBindingError()
	}
	if p.current.Type == lexer.TokenColon {
		return nil, p.unsupportedTypeAnnotationError()
	}

	var value ast.Expression
	if p.match(lexer.TokenAssign) {
		expr, err := p.parseSequence()
		if err != nil {
			return nil, err
		}
		value = expr
	} else if kind == ast.DeclarationConst || name == "" {
		return nil, errorAtToken(start, "%s declaration requires an initializer", kind)
	}

	if consumeTerminator {
		if err := p.consumeStatementTerminator(); err != nil {
			return nil, err
		}
	}
	return &ast.VariableDecl{
		BaseNode: baseFrom(start),
		Kind:     kind,
		Name:     name,
		Pattern:  pattern,
		Value:    value,
	}, nil
}

func declarationKind(tokenType lexer.TokenType) ast.DeclarationKind {
	if tokenType == lexer.TokenConst {
		return ast.DeclarationConst
	}
	return ast.DeclarationVar
}
