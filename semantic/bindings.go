package semantic

import "jayess-go/ast"

func declareBindingPattern(scope *scope, kind ast.DeclarationKind, pattern ast.BindingPattern) bool {
	_, ok := declareBindingPatternWithDuplicate(scope, kind, pattern)
	return ok
}

func declareBindingPatternWithDuplicate(scope *scope, kind ast.DeclarationKind, pattern ast.BindingPattern) (string, bool) {
	switch pattern := pattern.(type) {
	case nil:
		return "", true
	case *ast.BindingName:
		if kind == ast.DeclarationConst {
			return pattern.Name, scope.declareConst(pattern.Name)
		}
		return pattern.Name, scope.declare(pattern.Name)
	case *ast.BindingDefault:
		return declareBindingPatternWithDuplicate(scope, kind, pattern.Pattern)
	case *ast.BindingRest:
		return declareBindingPatternWithDuplicate(scope, kind, pattern.Pattern)
	case *ast.ArrayBindingPattern:
		for _, element := range pattern.Elements {
			if name, ok := declareBindingPatternWithDuplicate(scope, kind, element); !ok {
				return name, false
			}
		}
		return "", true
	case *ast.ObjectBindingPattern:
		for _, property := range pattern.Properties {
			if name, ok := declareBindingPatternWithDuplicate(scope, kind, property.Pattern); !ok {
				return name, false
			}
		}
		return "", true
	default:
		return "", false
	}
}

func analyzeBindingDefaults(scope *scope, pattern ast.BindingPattern) error {
	return analyzeBindingDefaultsWithContext(scope, rootContext(), pattern)
}

func analyzeBindingDefaultsWithContext(scope *scope, context controlContext, pattern ast.BindingPattern) error {
	switch pattern := pattern.(type) {
	case *ast.BindingDefault:
		if err := analyzeExpressionWithContext(scope, context, pattern.Value); err != nil {
			return err
		}
		return analyzeBindingDefaultsWithContext(scope, context, pattern.Pattern)
	case *ast.BindingRest:
		return analyzeBindingDefaultsWithContext(scope, context, pattern.Pattern)
	case *ast.ArrayBindingPattern:
		for _, element := range pattern.Elements {
			if err := analyzeBindingDefaultsWithContext(scope, context, element); err != nil {
				return err
			}
		}
	case *ast.ObjectBindingPattern:
		for _, property := range pattern.Properties {
			if err := analyzeOptionalExpressionWithContext(scope, context, property.KeyExpr); err != nil {
				return err
			}
			if err := analyzeBindingDefaultsWithContext(scope, context, property.Pattern); err != nil {
				return err
			}
		}
	}
	return nil
}
