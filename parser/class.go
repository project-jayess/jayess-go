package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseClassDeclaration() (ast.Statement, error) {
	return p.parseClassDeclarationWithName(true)
}

func (p *Parser) parseClassDeclarationWithName(requireName bool) (ast.Statement, error) {
	start := p.current
	p.advance()
	name := ""
	if p.current.Type == lexer.TokenIdent {
		name = p.current.Literal
		p.advance()
		if p.current.Type == lexer.TokenLt {
			return nil, p.unsupportedGenericTypeParametersError()
		}
	} else if requireName {
		return nil, p.errorAtCurrent("expected class name, got %s", p.current.Type)
	}
	var superClass ast.Expression
	if p.match(lexer.TokenExtends) {
		var err error
		superClass, err = p.parsePostfix()
		if err != nil {
			return nil, err
		}
	}
	if p.isUnsupportedImplementsClauseStart() {
		return nil, p.unsupportedImplementsClauseError()
	}
	if err := p.expect(lexer.TokenLBrace); err != nil {
		return nil, err
	}
	members := []ast.ClassMember{}
	for !p.match(lexer.TokenRBrace) {
		if p.current.Type == lexer.TokenEOF {
			return nil, p.errorAtCurrent("expected class member or }, got %s", p.current.Type)
		}
		if p.match(lexer.TokenSemicolon) {
			continue
		}
		member, err := p.parseClassMember()
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return &ast.ClassDecl{
		BaseNode:   baseFrom(start),
		Name:       name,
		SuperClass: superClass,
		Members:    members,
	}, nil
}

func (p *Parser) parseClassMember() (ast.ClassMember, error) {
	start := p.current
	if p.current.Type == lexer.TokenAt {
		return ast.ClassMember{}, p.unsupportedDecoratorError()
	}
	if p.isClassStaticBlockStart() {
		return p.parseClassStaticBlock(start)
	}
	static := p.matchClassStaticModifier()
	if p.isUnsupportedAbstractClassMemberStart() {
		return ast.ClassMember{}, p.unsupportedAbstractModifierError()
	}
	if p.isUnsupportedClassAccessModifierStart() {
		return ast.ClassMember{}, p.unsupportedClassAccessModifierError()
	}
	if p.isUnsupportedReadonlyClassMemberStart() {
		return ast.ClassMember{}, p.unsupportedReadonlyModifierError()
	}
	if p.isUnsupportedOverrideClassMemberStart() {
		return ast.ClassMember{}, p.unsupportedOverrideModifierError()
	}
	if p.isUnsupportedAccessorClassMemberStart() {
		return ast.ClassMember{}, p.unsupportedAccessorModifierError()
	}
	isAsync := p.matchClassAsyncModifier()
	isGenerator := p.match(lexer.TokenStar)
	private := p.match(lexer.TokenHash)
	if p.current.Type == lexer.TokenLBracket {
		if private {
			return ast.ClassMember{}, p.errorAtCurrent("private computed class members are not supported")
		}
		return p.parseComputedClassMember(start, static, isAsync, isGenerator)
	}
	name := p.current
	if !isObjectPropertyNameToken(name.Type) {
		return ast.ClassMember{}, p.errorAtCurrent("expected class member name, got %s", p.current.Type)
	}
	p.advance()
	if p.current.Type == lexer.TokenQuestion {
		return ast.ClassMember{}, p.unsupportedOptionalPropertyError()
	}
	if !isAsync && !isGenerator && isAccessorKeyword(name.Literal) && (isObjectPropertyNameToken(p.current.Type) || p.current.Type == lexer.TokenHash || p.current.Type == lexer.TokenLBracket) {
		return p.parseClassAccessor(start, name.Literal, static, private)
	}
	if p.current.Type == lexer.TokenLt {
		return ast.ClassMember{}, p.unsupportedGenericTypeParametersError()
	}
	if p.current.Type != lexer.TokenLParen {
		if isAsync || isGenerator {
			return ast.ClassMember{}, p.errorAtCurrent("expected class method parameters, got %s", p.current.Type)
		}
		return p.parseClassField(start, name.Literal, static, private)
	}
	params, err := p.parseParameterList()
	if err != nil {
		return ast.ClassMember{}, err
	}
	if p.current.Type == lexer.TokenSemicolon {
		return ast.ClassMember{}, p.unsupportedClassMethodOverloadDeclarationError()
	}
	if p.current.Type == lexer.TokenColon {
		return ast.ClassMember{}, p.unsupportedReturnAnnotationError()
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return ast.ClassMember{}, err
	}
	memberName := name.Literal
	return ast.ClassMember{
		BaseNode:    baseFrom(start),
		Name:        memberName,
		Params:      params,
		Body:        body,
		Constructor: !static && memberName == "constructor",
		Private:     private,
		Static:      static,
		IsAsync:     isAsync,
		IsGenerator: isGenerator,
	}, nil
}
