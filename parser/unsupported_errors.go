package parser

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
