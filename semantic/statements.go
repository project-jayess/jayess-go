package semantic

import "jayess-go/ast"

func analyzeStatement(scope *scope, context controlContext, statement ast.Statement) error {
	switch stmt := statement.(type) {
	case *ast.EmptyStatement:
		return nil
	case *ast.DebuggerStatement:
		return nil
	case *ast.ImportDecl:
		return analyzeImportDeclaration(scope, stmt)
	case *ast.ExportDecl:
		return analyzeExportDeclaration(scope, context, stmt)
	case *ast.VariableDecl:
		if stmt.Value != nil {
			if err := analyzeExpressionWithContext(scope, context, stmt.Value); err != nil {
				return err
			}
		}
		if err := analyzeBindingDefaultsWithContext(scope, context, stmt.Pattern); err != nil {
			return err
		}
		if duplicate, ok := declareVariable(scope, stmt); !ok {
			return duplicateDeclarationError(stmt, duplicate)
		}
	case *ast.FunctionDecl:
		if !scope.declareConstructable(stmt.Name) {
			return errorAt(stmt, "duplicate declaration %s", stmt.Name)
		}
		return analyzeFunctionBody(scope, stmt.Params, stmt.Body, stmt.IsAsync, stmt.IsGenerator)
	case *ast.ClassDecl:
		return analyzeClassDeclaration(scope, stmt)
	case *ast.BlockStatement:
		return analyzeStatements(newScope(scope), context, stmt.Statements)
	case *ast.ExpressionStatement:
		return analyzeExpressionWithContext(scope, context, stmt.Expression)
	case *ast.AssignmentStatement:
		if err := analyzeAssignmentTarget(scope, context, stmt.Target); err != nil {
			return err
		}
		return analyzeExpressionWithContext(scope, context, stmt.Value)
	case *ast.ReturnStatement:
		if !context.inFunction {
			return errorAt(stmt, "return outside function")
		}
		return analyzeOptionalExpressionWithContext(scope, context, stmt.Value)
	case *ast.IfStatement:
		if err := analyzeExpressionWithContext(scope, context, stmt.Condition); err != nil {
			return err
		}
		if err := analyzeStatements(newScope(scope), context, stmt.Consequence); err != nil {
			return err
		}
		return analyzeStatements(newScope(scope), context, stmt.Alternative)
	case *ast.WhileStatement:
		if err := analyzeExpressionWithContext(scope, context, stmt.Condition); err != nil {
			return err
		}
		return analyzeStatements(newScope(scope), context.enterLoop(), stmt.Body)
	case *ast.DoWhileStatement:
		if err := analyzeStatements(newScope(scope), context.enterLoop(), stmt.Body); err != nil {
			return err
		}
		return analyzeExpressionWithContext(scope, context, stmt.Condition)
	case *ast.ForStatement:
		return analyzeForStatement(scope, context, stmt)
	case *ast.ForOfStatement:
		return analyzeForOfStatement(scope, context, stmt)
	case *ast.ForInStatement:
		return analyzeForInStatement(scope, context, stmt)
	case *ast.LabeledStatement:
		labelContext, ok := context.enterLabel(stmt.Label, stmt.Statement)
		if !ok {
			return errorAt(stmt, "duplicate label %s", stmt.Label)
		}
		return analyzeStatement(scope, labelContext, stmt.Statement)
	case *ast.SwitchStatement:
		return analyzeSwitchStatement(scope, context, stmt)
	case *ast.BreakStatement:
		if stmt.Label != "" {
			if _, ok := context.findLabel(stmt.Label); !ok {
				return errorAt(stmt, "unknown label %s", stmt.Label)
			}
			return nil
		}
		if !context.inLoop && !context.inSwitch {
			return errorAt(stmt, "break outside loop or switch")
		}
	case *ast.ContinueStatement:
		if stmt.Label != "" {
			label, ok := context.findLabel(stmt.Label)
			if !ok {
				return errorAt(stmt, "unknown label %s", stmt.Label)
			}
			if !label.allowsContinue {
				return errorAt(stmt, "continue target %s is not a loop", stmt.Label)
			}
			return nil
		}
		if !context.inLoop {
			return errorAt(stmt, "continue outside loop")
		}
	case *ast.ThrowStatement:
		return analyzeExpressionWithContext(scope, context, stmt.Value)
	case *ast.TryStatement:
		return analyzeTryStatement(scope, context, stmt)
	default:
		return nil
	}
	return nil
}

func declareVariable(scope *scope, stmt *ast.VariableDecl) (string, bool) {
	return declareBindingPatternWithDuplicate(scope, stmt.Kind, stmt.Pattern)
}

func duplicateDeclarationError(node ast.Node, name string) error {
	if name == "" {
		return errorAt(node, "duplicate declaration")
	}
	return errorAt(node, "duplicate declaration %s", name)
}

func analyzeStatements(scope *scope, context controlContext, statements []ast.Statement) error {
	for _, statement := range statements {
		if err := analyzeStatement(scope, context, statement); err != nil {
			return err
		}
	}
	return nil
}

func analyzeOptionalExpression(scope *scope, expr ast.Expression) error {
	return analyzeOptionalExpressionWithContext(scope, rootContext(), expr)
}

func analyzeOptionalExpressionWithContext(scope *scope, context controlContext, expr ast.Expression) error {
	if expr == nil {
		return nil
	}
	return analyzeExpressionWithContext(scope, context, expr)
}
