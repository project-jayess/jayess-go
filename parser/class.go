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

func (p *Parser) isClassStaticBlockStart() bool {
	if p.current.Type != lexer.TokenStatic {
		return false
	}

	state := p.snapshot()
	p.advance()
	next := p.current.Type
	p.restore(state)
	return next == lexer.TokenLBrace
}

func (p *Parser) parseClassStaticBlock(start lexer.Token) (ast.ClassMember, error) {
	p.advance()
	body, err := p.parseBlockStatements()
	if err != nil {
		return ast.ClassMember{}, err
	}
	return ast.ClassMember{
		BaseNode:    baseFrom(start),
		Body:        body,
		Static:      true,
		StaticBlock: true,
	}, nil
}

func (p *Parser) matchClassStaticModifier() bool {
	if p.current.Type != lexer.TokenStatic {
		return false
	}

	state := p.snapshot()
	p.advance()
	next := p.current.Type
	p.restore(state)

	if next == lexer.TokenHash || next == lexer.TokenLBracket || isObjectPropertyNameToken(next) {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) matchClassAsyncModifier() bool {
	if p.current.Type != lexer.TokenAsync {
		return false
	}

	state := p.snapshot()
	start := p.current
	p.advance()
	if p.current.Line > start.Line {
		p.restore(state)
		return false
	}
	next := p.current.Type
	p.restore(state)

	if next == lexer.TokenStar || next == lexer.TokenHash || next == lexer.TokenLBracket || isObjectPropertyNameToken(next) {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) parseComputedClassMember(start lexer.Token, static bool, isAsync bool, isGenerator bool) (ast.ClassMember, error) {
	key, err := p.parseComputedClassKey()
	if err != nil {
		return ast.ClassMember{}, err
	}
	if p.current.Type == lexer.TokenQuestion {
		return ast.ClassMember{}, p.unsupportedOptionalPropertyError()
	}
	if p.current.Type != lexer.TokenLParen {
		if isAsync || isGenerator {
			return ast.ClassMember{}, p.errorAtCurrent("expected computed class method parameters, got %s", p.current.Type)
		}
		return p.parseClassFieldWithKey(start, "", key, true, static, false)
	}
	params, err := p.parseParameterList()
	if err != nil {
		return ast.ClassMember{}, err
	}
	if p.current.Type == lexer.TokenColon {
		return ast.ClassMember{}, p.unsupportedReturnAnnotationError()
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return ast.ClassMember{}, err
	}
	return ast.ClassMember{
		BaseNode:    baseFrom(start),
		KeyExpr:     key,
		Params:      params,
		Body:        body,
		Computed:    true,
		Static:      static,
		IsAsync:     isAsync,
		IsGenerator: isGenerator,
	}, nil
}

func (p *Parser) parseClassAccessor(start lexer.Token, kind string, static bool, private bool) (ast.ClassMember, error) {
	if !private {
		private = p.match(lexer.TokenHash)
	}
	if p.current.Type == lexer.TokenLBracket {
		if private {
			return ast.ClassMember{}, p.errorAtCurrent("private computed class accessors are not supported")
		}
		return p.parseComputedClassAccessor(start, kind, static)
	}
	name := p.current
	if !isObjectPropertyNameToken(name.Type) {
		return ast.ClassMember{}, p.errorAtCurrent("expected class accessor name, got %s", p.current.Type)
	}
	p.advance()
	params, err := p.parseParameterList()
	if err != nil {
		return ast.ClassMember{}, err
	}
	if err := validateNamedAccessorParameters(kind, name, params); err != nil {
		return ast.ClassMember{}, err
	}
	if p.current.Type == lexer.TokenColon {
		return ast.ClassMember{}, p.unsupportedReturnAnnotationError()
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return ast.ClassMember{}, err
	}
	return ast.ClassMember{
		BaseNode: baseFrom(start),
		Name:     name.Literal,
		Params:   params,
		Body:     body,
		Getter:   kind == "get",
		Setter:   kind == "set",
		Private:  private,
		Static:   static,
	}, nil
}

func (p *Parser) parseComputedClassAccessor(start lexer.Token, kind string, static bool) (ast.ClassMember, error) {
	key, err := p.parseComputedClassKey()
	if err != nil {
		return ast.ClassMember{}, err
	}
	params, err := p.parseParameterList()
	if err != nil {
		return ast.ClassMember{}, err
	}
	if err := validateComputedAccessorParameters(kind, params, p.errorAtCurrent); err != nil {
		return ast.ClassMember{}, err
	}
	if p.current.Type == lexer.TokenColon {
		return ast.ClassMember{}, p.unsupportedReturnAnnotationError()
	}
	body, err := p.parseBlockStatements()
	if err != nil {
		return ast.ClassMember{}, err
	}
	return ast.ClassMember{
		BaseNode: baseFrom(start),
		KeyExpr:  key,
		Params:   params,
		Body:     body,
		Computed: true,
		Getter:   kind == "get",
		Setter:   kind == "set",
		Static:   static,
	}, nil
}

func (p *Parser) parseComputedClassKey() (ast.Expression, error) {
	if err := p.expect(lexer.TokenLBracket); err != nil {
		return nil, err
	}
	key, err := p.parseConditional()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TokenRBracket); err != nil {
		return nil, err
	}
	return key, nil
}

func isAccessorKeyword(name string) bool {
	return name == "get" || name == "set"
}

func (p *Parser) parseClassField(start lexer.Token, name string, static bool, private bool) (ast.ClassMember, error) {
	return p.parseClassFieldWithKey(start, name, nil, false, static, private)
}

func (p *Parser) parseClassFieldWithKey(start lexer.Token, name string, key ast.Expression, computed bool, static bool, private bool) (ast.ClassMember, error) {
	var value ast.Expression
	var err error
	if p.current.Type == lexer.TokenBang {
		return ast.ClassMember{}, p.unsupportedDefiniteAssignmentAssertionError()
	}
	if p.current.Type == lexer.TokenColon {
		return ast.ClassMember{}, p.unsupportedTypeAnnotationError()
	}
	if p.match(lexer.TokenAssign) {
		value, err = p.parseSequence()
		if err != nil {
			return ast.ClassMember{}, err
		}
	}
	if err := p.consumeStatementTerminator(); err != nil {
		return ast.ClassMember{}, err
	}
	return ast.ClassMember{
		BaseNode: baseFrom(start),
		Name:     name,
		KeyExpr:  key,
		Value:    value,
		Computed: computed,
		Field:    true,
		Private:  private,
		Static:   static,
	}, nil
}
