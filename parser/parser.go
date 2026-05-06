package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

type Parser struct {
	lexer    *lexer.Lexer
	previous lexer.Token
	current  lexer.Token
}

type parserState struct {
	lexer    lexer.State
	previous lexer.Token
	current  lexer.Token
}

func New(l *lexer.Lexer) *Parser {
	return &Parser{lexer: l, current: l.NextToken()}
}

func (p *Parser) ParseProgram() (*ast.Program, error) {
	program := &ast.Program{BaseNode: baseFrom(p.current)}
	for p.current.Type != lexer.TokenEOF {
		statement, err := p.ParseStatement()
		if err != nil {
			return nil, err
		}
		program.Statements = append(program.Statements, statement)
	}
	return program, nil
}

func (p *Parser) ParseExpression() (ast.Expression, error) {
	expr, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if p.current.Type != lexer.TokenEOF {
		return nil, p.errorAtCurrent("expected end of expression, got %s", p.current.Type)
	}
	return expr, nil
}

func (p *Parser) advance() {
	p.previous = p.current
	p.current = p.lexer.NextToken()
}

func (p *Parser) snapshot() parserState {
	return parserState{
		lexer:    p.lexer.Snapshot(),
		previous: p.previous,
		current:  p.current,
	}
}

func (p *Parser) restore(state parserState) {
	p.lexer.Restore(state.lexer)
	p.previous = state.previous
	p.current = state.current
}

func (p *Parser) match(tokenType lexer.TokenType) bool {
	if p.current.Type != tokenType {
		return false
	}
	p.advance()
	return true
}

func (p *Parser) expect(tokenType lexer.TokenType) error {
	if p.current.Type != tokenType {
		if p.current.Type == lexer.TokenEOF {
			return p.errorAtCurrent("expected %s before end of file", tokenType)
		}
		return p.errorAtCurrent("expected %s, got %s", tokenType, p.current.Type)
	}
	p.advance()
	return nil
}

func baseFrom(token lexer.Token) ast.BaseNode {
	return ast.BaseNode{Pos: ast.SourcePos{Line: token.Line, Column: token.Column}}
}

func baseOf(expr ast.Expression) ast.BaseNode {
	return ast.BaseNode{Pos: ast.PositionOf(expr)}
}
