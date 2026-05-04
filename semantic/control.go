package semantic

import "jayess-go/ast"

func analyzeForStatement(parent *scope, context controlContext, stmt *ast.ForStatement) error {
	loopScope := newScope(parent)
	if stmt.Init != nil {
		if err := analyzeStatement(loopScope, context, stmt.Init); err != nil {
			return err
		}
	}
	if err := analyzeOptionalExpressionWithContext(loopScope, context, stmt.Condition); err != nil {
		return err
	}
	if stmt.Update != nil {
		if err := analyzeStatement(loopScope, context, stmt.Update); err != nil {
			return err
		}
	}
	return analyzeStatements(newScope(loopScope), context.enterLoop(), stmt.Body)
}

func analyzeForOfStatement(parent *scope, context controlContext, stmt *ast.ForOfStatement) error {
	if stmt.Await && !context.inAsyncFunction {
		return errorAt(stmt, "for await outside async function")
	}
	if err := analyzeExpressionWithContext(parent, context, stmt.Iterable); err != nil {
		return err
	}
	if stmt.Target != nil {
		if err := analyzeAssignmentTarget(parent, context, stmt.Target); err != nil {
			return err
		}
		return analyzeStatements(newScope(parent), context.enterLoop(), stmt.Body)
	}
	if err := analyzeBindingDefaultsWithContext(parent, context, stmt.Pattern); err != nil {
		return err
	}
	loopScope := newScope(parent)
	if duplicate, ok := declareForOfVariable(loopScope, stmt); !ok {
		return duplicateForBindingError(stmt, "for...of", duplicate)
	}
	return analyzeStatements(newScope(loopScope), context.enterLoop(), stmt.Body)
}

func analyzeForInStatement(parent *scope, context controlContext, stmt *ast.ForInStatement) error {
	if err := analyzeExpressionWithContext(parent, context, stmt.Object); err != nil {
		return err
	}
	if stmt.Target != nil {
		if err := analyzeAssignmentTarget(parent, context, stmt.Target); err != nil {
			return err
		}
		return analyzeStatements(newScope(parent), context.enterLoop(), stmt.Body)
	}
	if err := analyzeBindingDefaultsWithContext(parent, context, stmt.Pattern); err != nil {
		return err
	}
	loopScope := newScope(parent)
	if duplicate, ok := declareForInVariable(loopScope, stmt); !ok {
		return duplicateForBindingError(stmt, "for...in", duplicate)
	}
	return analyzeStatements(newScope(loopScope), context.enterLoop(), stmt.Body)
}

func declareForOfVariable(scope *scope, stmt *ast.ForOfStatement) (string, bool) {
	return declareBindingPatternWithDuplicate(scope, stmt.Kind, stmt.Pattern)
}

func declareForInVariable(scope *scope, stmt *ast.ForInStatement) (string, bool) {
	return declareBindingPatternWithDuplicate(scope, stmt.Kind, stmt.Pattern)
}

func duplicateForBindingError(node ast.Node, kind string, name string) error {
	if name == "" {
		return errorAt(node, "duplicate %s binding", kind)
	}
	return errorAt(node, "duplicate %s binding %s", kind, name)
}

func analyzeSwitchStatement(parent *scope, context controlContext, stmt *ast.SwitchStatement) error {
	if err := analyzeExpressionWithContext(parent, context, stmt.Discriminant); err != nil {
		return err
	}
	if err := validateSwitchDeclarations(stmt); err != nil {
		return err
	}
	switchScope := newScope(parent)
	switchContext := context.enterSwitch()
	for _, switchCase := range stmt.Cases {
		if err := analyzeExpressionWithContext(switchScope, context, switchCase.Test); err != nil {
			return err
		}
		if err := analyzeStatements(newScope(switchScope), switchContext, switchCase.Consequent); err != nil {
			return err
		}
	}
	return analyzeStatements(newScope(switchScope), switchContext, stmt.Default)
}

func validateSwitchDeclarations(stmt *ast.SwitchStatement) error {
	switchScope := newScope(nil)
	for _, switchCase := range stmt.Cases {
		if err := declareSwitchConsequentBindings(switchScope, switchCase.Consequent); err != nil {
			return err
		}
	}
	return declareSwitchConsequentBindings(switchScope, stmt.Default)
}

func declareSwitchConsequentBindings(scope *scope, statements []ast.Statement) error {
	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *ast.VariableDecl:
			if duplicate, ok := declareBindingPatternWithDuplicate(scope, stmt.Kind, stmt.Pattern); !ok {
				return duplicateDeclarationError(stmt, duplicate)
			}
		case *ast.FunctionDecl:
			if !scope.declareConstructable(stmt.Name) {
				return errorAt(stmt, "duplicate declaration %s", stmt.Name)
			}
		case *ast.ClassDecl:
			if stmt.Name != "" && !scope.declareConstructable(stmt.Name) {
				return errorAt(stmt, "duplicate declaration %s", stmt.Name)
			}
		}
	}
	return nil
}

func analyzeTryStatement(parent *scope, context controlContext, stmt *ast.TryStatement) error {
	if err := analyzeStatements(newScope(parent), context, stmt.TryBody); err != nil {
		return err
	}
	catchScope := newScope(parent)
	if stmt.CatchPattern != nil {
		if err := analyzeBindingDefaultsWithContext(catchScope, context, stmt.CatchPattern); err != nil {
			return err
		}
		if duplicate, ok := declareBindingPatternWithDuplicate(catchScope, ast.DeclarationVar, stmt.CatchPattern); !ok {
			return duplicateDeclarationError(stmt, duplicate)
		}
	}
	if err := analyzeStatements(catchScope, context, stmt.CatchBody); err != nil {
		return err
	}
	return analyzeStatements(newScope(parent), context, stmt.FinallyBody)
}
