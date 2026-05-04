package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseUnsupportedLetDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("let declarations are not supported; use var or const")
}

func (p *Parser) parseUnsupportedPublicModifier() (ast.Statement, error) {
	return nil, p.errorAtCurrent("public is not supported; use export for module visibility")
}

func (p *Parser) parseUnsupportedTopLevelPrivate() (ast.Statement, error) {
	return nil, p.errorAtCurrent("top-level private is not supported; use #members inside classes")
}

func (p *Parser) parseUnsupportedWithStatement() (ast.Statement, error) {
	return nil, p.errorAtCurrent("with statements are not supported")
}

func (p *Parser) parseUnsupportedEnumDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("enum declarations are not supported")
}

func (p *Parser) parseUnsupportedUsingDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("using declarations are not supported")
}

func (p *Parser) parseUnsupportedAbstractModifier() (ast.Statement, error) {
	return nil, p.unsupportedAbstractModifierError()
}

func (p *Parser) parseUnsupportedDecorator() (ast.Statement, error) {
	return nil, p.unsupportedDecoratorError()
}

func (p *Parser) parseUnsupportedTypeAlias() (ast.Statement, error) {
	return nil, p.errorAtCurrent("type aliases are not supported")
}

func (p *Parser) parseUnsupportedInterfaceDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("interface declarations are not supported")
}

func (p *Parser) parseUnsupportedAmbientDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("ambient declarations are not supported")
}

func (p *Parser) parseUnsupportedNamespaceDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("namespace declarations are not supported")
}

func (p *Parser) parseUnsupportedModuleDeclaration() (ast.Statement, error) {
	return nil, p.errorAtCurrent("module declarations are not supported")
}

func (p *Parser) isUnsupportedTypeAliasStart() bool {
	if p.current.Literal != "type" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type != lexer.TokenIdent {
		return false
	}
	p.advance()
	return p.current.Type == lexer.TokenAssign
}

func (p *Parser) isUnsupportedConstEnumDeclarationStart() bool {
	if p.current.Type != lexer.TokenConst {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return p.current.Type == lexer.TokenEnum
}

func (p *Parser) isUnsupportedInterfaceStart() bool {
	if p.current.Literal != "interface" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type != lexer.TokenIdent {
		return false
	}
	p.advance()
	return p.current.Type == lexer.TokenLBrace
}

func (p *Parser) isUnsupportedAmbientDeclarationStart() bool {
	return p.current.Type == lexer.TokenIdent && p.current.Literal == "declare"
}

func (p *Parser) isUnsupportedNamespaceDeclarationStart() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "namespace" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type != lexer.TokenIdent {
		return false
	}
	p.advance()
	return p.current.Type == lexer.TokenLBrace
}

func (p *Parser) isUnsupportedModuleDeclarationStart() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "module" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type != lexer.TokenIdent && p.current.Type != lexer.TokenString {
		return false
	}
	p.advance()
	return p.current.Type == lexer.TokenLBrace
}

func (p *Parser) isUnsupportedImportEqualsDeclarationStart() bool {
	if p.current.Type != lexer.TokenIdent {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return p.current.Type == lexer.TokenAssign
}

func (p *Parser) isUnsupportedExportAsNamespaceDeclarationStart() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "as" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return p.current.Type == lexer.TokenIdent && p.current.Literal == "namespace"
}

func (p *Parser) isUnsupportedTypeOnlyModuleSpecifierStart() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "type" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type == lexer.TokenIdent && p.current.Literal == "as" {
		return false
	}
	return isModuleSpecifierNameToken(p.current.Type)
}

func (p *Parser) isUnsupportedUsingDeclarationStart() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "using" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return isUsingDeclarationBindingStart(p.current.Type)
}

func (p *Parser) isUnsupportedAwaitUsingDeclarationStart() bool {
	if p.current.Type != lexer.TokenAwait {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "using" {
		return false
	}
	p.advance()
	return isUsingDeclarationBindingStart(p.current.Type)
}

func isUsingDeclarationBindingStart(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenIdent, lexer.TokenLBrace, lexer.TokenLBracket:
		return true
	default:
		return false
	}
}

func (p *Parser) isUnsupportedImplementsClauseStart() bool {
	return p.current.Type == lexer.TokenIdent && p.current.Literal == "implements"
}

func (p *Parser) isUnsupportedAbstractClassDeclarationStart() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "abstract" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return p.current.Type == lexer.TokenClass
}

func (p *Parser) isUnsupportedReadonlyClassMemberStart() bool {
	return p.isUnsupportedClassMemberModifierStart("readonly")
}

func (p *Parser) isUnsupportedAbstractClassMemberStart() bool {
	return p.isUnsupportedClassMemberModifierStart("abstract")
}

func (p *Parser) isUnsupportedClassAccessModifierStart() bool {
	return p.isUnsupportedClassMemberModifierStart("public") ||
		p.isUnsupportedClassMemberModifierStart("private") ||
		p.isUnsupportedClassMemberModifierStart("protected")
}

func (p *Parser) isUnsupportedOverrideClassMemberStart() bool {
	return p.isUnsupportedClassMemberModifierStart("override")
}

func (p *Parser) isUnsupportedAccessorClassMemberStart() bool {
	return p.isUnsupportedClassMemberModifierStart("accessor")
}

func (p *Parser) isUnsupportedClassMemberModifierStart(name string) bool {
	if p.current.Literal != name {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return isClassMemberModifierTargetStart(p.current.Type)
}

func isClassMemberModifierTargetStart(tokenType lexer.TokenType) bool {
	return tokenType == lexer.TokenHash || tokenType == lexer.TokenLBracket || isObjectPropertyNameToken(tokenType)
}

func (p *Parser) isUnsupportedParameterPropertyModifierStart() bool {
	return p.isUnsupportedParameterModifierStart("public") ||
		p.isUnsupportedParameterModifierStart("private") ||
		p.isUnsupportedParameterModifierStart("protected") ||
		p.isUnsupportedParameterModifierStart("readonly")
}

func (p *Parser) isUnsupportedParameterModifierStart(name string) bool {
	if p.current.Literal != name {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return isParameterPropertyTargetStart(p.current.Type)
}

func isParameterPropertyTargetStart(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenIdent, lexer.TokenLBrace, lexer.TokenLBracket:
		return true
	default:
		return false
	}
}

func (p *Parser) isUnsupportedConstAssertionSuffix() bool {
	if p.current.Type != lexer.TokenIdent || p.current.Literal != "as" {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	return p.current.Type == lexer.TokenConst
}

func (p *Parser) isUnsupportedReturnTypePredicateStart() bool {
	if p.current.Type != lexer.TokenColon {
		return false
	}
	state := p.snapshot()
	defer p.restore(state)

	p.advance()
	if p.current.Type == lexer.TokenIdent && p.current.Literal == "asserts" {
		return true
	}
	if p.current.Type != lexer.TokenIdent && p.current.Type != lexer.TokenThis {
		return false
	}
	p.advance()
	return p.current.Type == lexer.TokenIs
}

func (p *Parser) unsupportedTypeAnnotationError() error {
	return p.errorAtCurrent("type annotations are not supported")
}

func (p *Parser) unsupportedReturnAnnotationError() error {
	if p.isUnsupportedReturnTypePredicateStart() {
		return p.unsupportedReturnTypePredicateError()
	}
	return p.unsupportedReturnTypeAnnotationError()
}

func (p *Parser) unsupportedReturnTypeAnnotationError() error {
	return p.errorAtCurrent("return type annotations are not supported")
}

func (p *Parser) unsupportedReturnTypePredicateError() error {
	return p.errorAtCurrent("type predicate and assertion return annotations are not supported")
}

func (p *Parser) unsupportedFunctionOverloadDeclarationError() error {
	return p.errorAtCurrent("function overload declarations are not supported")
}

func (p *Parser) unsupportedClassMethodOverloadDeclarationError() error {
	return p.errorAtCurrent("class method overload declarations are not supported")
}

func (p *Parser) unsupportedParameterPropertyModifierError() error {
	return p.errorAtCurrent("parameter property modifiers are not supported")
}

func (p *Parser) unsupportedOptionalParameterError() error {
	return p.errorAtCurrent("optional parameters are not supported")
}

func (p *Parser) unsupportedOptionalPropertyError() error {
	return p.errorAtCurrent("optional properties and methods are not supported")
}

func (p *Parser) unsupportedOptionalBindingError() error {
	return p.errorAtCurrent("optional bindings are not supported")
}

func (p *Parser) unsupportedGenericTypeParametersError() error {
	return p.errorAtCurrent("generic type parameters are not supported")
}

func (p *Parser) unsupportedImplementsClauseError() error {
	return p.errorAtCurrent("implements clauses are not supported")
}

func (p *Parser) unsupportedAbstractModifierError() error {
	return p.errorAtCurrent("abstract modifiers are not supported")
}

func (p *Parser) unsupportedReadonlyModifierError() error {
	return p.errorAtCurrent("readonly modifiers are not supported")
}

func (p *Parser) unsupportedClassAccessModifierError() error {
	return p.errorAtCurrent("class access modifiers are not supported; use #members for private fields and methods")
}

func (p *Parser) unsupportedOverrideModifierError() error {
	return p.errorAtCurrent("override modifiers are not supported")
}

func (p *Parser) unsupportedAccessorModifierError() error {
	return p.errorAtCurrent("accessor modifiers are not supported")
}

func (p *Parser) unsupportedTypeOnlyModuleDeclarationError() error {
	return p.errorAtCurrent("type-only imports and exports are not supported")
}

func (p *Parser) unsupportedImportEqualsDeclarationError() error {
	return p.errorAtCurrent("import equals declarations are not supported; use ESM imports")
}

func (p *Parser) unsupportedExportEqualsDeclarationError() error {
	return p.errorAtCurrent("export equals declarations are not supported; use ESM exports")
}

func (p *Parser) unsupportedExportAsNamespaceDeclarationError() error {
	return p.errorAtCurrent("export as namespace declarations are not supported; use ESM exports")
}

func (p *Parser) unsupportedImportAttributesError() error {
	return p.errorAtCurrent("import attributes are not supported")
}

func (p *Parser) unsupportedDynamicImportError() error {
	return p.errorAtCurrent("dynamic import expressions are not supported")
}

func (p *Parser) unsupportedAsyncLineTerminatorError() error {
	return p.errorAtCurrent("line terminator after async is not allowed before function or arrow function")
}

func (p *Parser) unsupportedArrowLineTerminatorError() error {
	return p.errorAtCurrent("line terminator before arrow is not allowed")
}

func (p *Parser) unsupportedTypeAssertionError() error {
	return p.errorAtCurrent("type assertions are not supported")
}

func (p *Parser) unsupportedConstAssertionError() error {
	return p.errorAtCurrent("const assertions are not supported")
}

func (p *Parser) unsupportedSatisfiesExpressionError() error {
	return p.errorAtCurrent("satisfies expressions are not supported")
}

func (p *Parser) unsupportedNonNullAssertionError() error {
	return p.errorAtCurrent("non-null assertions are not supported")
}

func (p *Parser) unsupportedDefiniteAssignmentAssertionError() error {
	return p.errorAtCurrent("definite assignment assertions are not supported")
}

func (p *Parser) unsupportedDecoratorError() error {
	return p.errorAtCurrent("decorators are not supported")
}

func (p *Parser) unsupportedRegularExpressionLiteralError() error {
	return p.errorAtCurrent("regular expression literals are not supported; use RegExp")
}

func (p *Parser) unsupportedJSXOrAngleBracketSyntaxError() error {
	return p.errorAtCurrent("JSX and angle-bracket type assertions are not supported")
}

func (p *Parser) missingSpreadExpressionError() error {
	return p.errorAtCurrent("spread element requires an expression")
}
