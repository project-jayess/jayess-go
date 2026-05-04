package semantic

import "jayess-go/ast"

func analyzeFunctionBody(parent *scope, params []ast.Parameter, body []ast.Statement, isAsync bool, isGenerator bool) error {
	return analyzeFunctionBodyWithContext(parent, params, body, rootContext().enterFunction(isAsync, isGenerator))
}

func analyzeFunctionBodyWithContext(parent *scope, params []ast.Parameter, body []ast.Statement, context controlContext) error {
	functionScope := newScope(parent)
	if err := declareParametersWithContext(functionScope, context, params); err != nil {
		return err
	}
	declareArgumentsBinding(functionScope, context)
	return analyzeStatements(functionScope, context, body)
}

func declareParameters(scope *scope, params []ast.Parameter) error {
	return declareParametersWithContext(scope, rootContext(), params)
}

func declareParametersWithContext(scope *scope, context controlContext, params []ast.Parameter) error {
	for _, param := range params {
		if param.Default != nil {
			if err := analyzeExpressionWithContext(scope, context, param.Default); err != nil {
				return err
			}
		}
		if err := analyzeBindingDefaultsWithContext(scope, context, param.Pattern); err != nil {
			return err
		}
		if duplicate, ok := declareBindingPatternWithDuplicate(scope, ast.DeclarationVar, param.Pattern); !ok {
			if duplicate == "" {
				return errorf("duplicate parameter")
			}
			return errorf("duplicate parameter %s", duplicate)
		}
	}
	return nil
}

func declareArgumentsBinding(scope *scope, context controlContext) {
	if context.ownsArguments && !scope.hasLocal("arguments") {
		scope.declare("arguments")
	}
}
