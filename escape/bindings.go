package escape

import "jayess-go/ast"

func declarePattern(names map[string]bool, pattern ast.BindingPattern) {
	switch pattern := pattern.(type) {
	case *ast.BindingName:
		names[pattern.Name] = true
	case *ast.BindingDefault:
		declarePattern(names, pattern.Pattern)
	case *ast.BindingRest:
		declarePattern(names, pattern.Pattern)
	case *ast.ArrayBindingPattern:
		for _, element := range pattern.Elements {
			declarePattern(names, element)
		}
	case *ast.ObjectBindingPattern:
		for _, property := range pattern.Properties {
			declarePattern(names, property.Pattern)
		}
	}
}

func declareParameters(names map[string]bool, params []ast.Parameter) {
	for _, param := range params {
		declarePattern(names, param.Pattern)
	}
}

func declarePatternInScope(scope *scope, pattern ast.BindingPattern) {
	names := map[string]bool{}
	declarePattern(names, pattern)
	for name := range names {
		scope.declare(name)
	}
}

func declareParametersInScope(scope *scope, params []ast.Parameter) {
	for _, param := range params {
		declarePatternInScope(scope, param.Pattern)
	}
}
