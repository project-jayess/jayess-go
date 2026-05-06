package semantic

import "jayess-go/ast"

func analyzeAssignmentTarget(scope *scope, context controlContext, target ast.Expression) error {
	switch target := target.(type) {
	case *ast.Identifier:
		bind, ok := scope.resolve(target.Name)
		if !ok {
			return errorAt(target, "assignment to %s before declaration", target.Name)
		}
		if !bind.mutable {
			return errorAt(target, "assignment to const %s; use var for mutable bindings", target.Name)
		}
		return nil
	case *ast.MemberExpression:
		if err := analyzeExpressionWithContext(scope, context, target.Target); err != nil {
			return err
		}
		if target.Private {
			return analyzePrivateMemberAccess(context, target)
		}
		return nil
	case *ast.IndexExpression:
		return analyzeExpressionsWithContext(scope, context, target.Target, target.Index)
	default:
		return errorAt(target, "invalid assignment target")
	}
}
