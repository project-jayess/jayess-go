package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func validateNamedAccessorParameters(kind string, name lexer.Token, params []ast.Parameter) error {
	if kind == "get" && len(params) != 0 {
		return errorAtToken(name, "getter %s cannot have parameters", name.Literal)
	}
	if kind == "set" {
		return validateSetterParameters(params, func(message string) error {
			return errorAtToken(name, "setter %s %s", name.Literal, message)
		})
	}
	return nil
}

func validateComputedAccessorParameters(kind string, params []ast.Parameter, errorAtCurrent func(string, ...any) error) error {
	if kind == "get" && len(params) != 0 {
		return errorAtCurrent("computed getter cannot have parameters")
	}
	if kind == "set" {
		return validateSetterParameters(params, func(message string) error {
			return errorAtCurrent("computed setter " + message)
		})
	}
	return nil
}

func validateSetterParameters(params []ast.Parameter, errorf func(string) error) error {
	if len(params) != 1 {
		return errorf("must have exactly one parameter")
	}
	if params[0].Rest {
		return errorf("cannot use a rest parameter")
	}
	return nil
}
