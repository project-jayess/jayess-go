package lowering

import (
	"fmt"
	"sync/atomic"

	"jayess-go/ast"
	"jayess-go/ir"
)

func inferIRKind(expr ir.Expression) ir.ValueKind {
	switch expr := expr.(type) {
	case *ir.NumberLiteral:
		return ir.ValueNumber
	case *ir.BigIntLiteral:
		return ir.ValueBigInt
	case *ir.BinaryExpression:
		if expr.Operator == ir.OperatorAdd && (inferIRKind(expr.Left) == ir.ValueString || inferIRKind(expr.Right) == ir.ValueString) {
			return ir.ValueString
		}
		return expr.Kind
	case *ir.BooleanLiteral, *ir.ComparisonExpression:
		return ir.ValueBoolean
	case *ir.NullLiteral, *ir.UndefinedLiteral:
		return ir.ValueDynamic
	case *ir.UnaryExpression, *ir.LogicalExpression:
		if unary, ok := expr.(*ir.UnaryExpression); ok {
			return unary.Kind
		}
		return ir.ValueBoolean
	case *ir.NullishCoalesceExpression:
		return expr.Kind
	case *ir.CommaExpression:
		return expr.Kind
	case *ir.ConditionalExpression:
		return expr.Kind
	case *ir.TypeofExpression:
		return ir.ValueString
	case *ir.NewTargetExpression:
		return expr.Kind
	case *ir.InstanceofExpression:
		return ir.ValueBoolean
	case *ir.StringLiteral:
		return ir.ValueString
	case *ir.IndexExpression:
		return expr.Kind
	case *ir.ArrayLiteral:
		return ir.ValueArray
	case *ir.TemplateLiteral:
		return ir.ValueString
	case *ir.SpreadExpression:
		return inferIRKind(expr.Value)
	case *ir.ObjectLiteral:
		return ir.ValueObject
	case *ir.FunctionValue:
		return ir.ValueFunction
	case *ir.MemberExpression:
		return ir.ValueDynamic
	case *ir.VariableRef:
		return expr.Kind
	case *ir.CallExpression:
		return expr.Kind
	case *ir.InvokeExpression:
		return expr.Kind
	default:
		return ""
	}
}

func lowerVisibility(visibility ast.Visibility) ir.Visibility {
	if visibility == ast.VisibilityPrivate {
		return ir.VisibilityPrivate
	}
	return ir.VisibilityPublic
}

func lowerDeclarationKind(kind ast.DeclarationKind) ir.DeclarationKind {
	switch kind {
	case ast.DeclarationConst:
		return ir.DeclarationConst
	case ast.DeclarationLet:
		return ir.DeclarationLet
	default:
		return ir.DeclarationVar
	}
}

func lowerOperator(op ast.BinaryOperator) ir.BinaryOperator {
	switch op {
	case ast.OperatorAdd:
		return ir.OperatorAdd
	case ast.OperatorSub:
		return ir.OperatorSub
	case ast.OperatorMul:
		return ir.OperatorMul
	case ast.OperatorDiv:
		return ir.OperatorDiv
	case ast.OperatorBitAnd:
		return ir.OperatorBitAnd
	case ast.OperatorBitOr:
		return ir.OperatorBitOr
	case ast.OperatorBitXor:
		return ir.OperatorBitXor
	case ast.OperatorShl:
		return ir.OperatorShl
	case ast.OperatorShr:
		return ir.OperatorShr
	default:
		return ir.OperatorUShr
	}
}

func lowerUnaryOperator(op ast.UnaryOperator) ir.UnaryOperator {
	if op == ast.OperatorBitNot {
		return ir.OperatorBitNot
	}
	return ir.OperatorNot
}

func lowerBinaryResultKind(op ast.BinaryOperator, left ir.ValueKind, right ir.ValueKind) ir.ValueKind {
	switch op {
	case ast.OperatorAdd:
		if left == ir.ValueString || right == ir.ValueString {
			return ir.ValueString
		}
		if left == ir.ValueDynamic || right == ir.ValueDynamic {
			return ir.ValueDynamic
		}
		return ir.ValueNumber
	case ast.OperatorSub, ast.OperatorMul, ast.OperatorDiv:
		return ir.ValueNumber
	case ast.OperatorBitAnd, ast.OperatorBitOr, ast.OperatorBitXor, ast.OperatorShl, ast.OperatorShr:
		if left == ir.ValueBigInt && right == ir.ValueBigInt {
			return ir.ValueBigInt
		}
		if left == ir.ValueDynamic || right == ir.ValueDynamic {
			if left == ir.ValueBigInt || right == ir.ValueBigInt {
				return ir.ValueDynamic
			}
			return ir.ValueNumber
		}
		return ir.ValueNumber
	case ast.OperatorUShr:
		if left == ir.ValueDynamic || right == ir.ValueDynamic {
			return ir.ValueDynamic
		}
		return ir.ValueNumber
	default:
		return ir.ValueDynamic
	}
}

func lowerUnaryResultKind(op ast.UnaryOperator, right ir.ValueKind) ir.ValueKind {
	switch op {
	case ast.OperatorBitNot:
		if right == ir.ValueBigInt {
			return ir.ValueBigInt
		}
		if right == ir.ValueDynamic {
			return ir.ValueDynamic
		}
		return ir.ValueNumber
	default:
		return ir.ValueBoolean
	}
}

func lowerComparisonOperator(op ast.ComparisonOperator) ir.ComparisonOperator {
	switch op {
	case ast.OperatorEq:
		return ir.OperatorEq
	case ast.OperatorNe:
		return ir.OperatorNe
	case ast.OperatorStrictEq:
		return ir.OperatorStrictEq
	case ast.OperatorStrictNe:
		return ir.OperatorStrictNe
	case ast.OperatorLt:
		return ir.OperatorLt
	case ast.OperatorLte:
		return ir.OperatorLte
	case ast.OperatorGt:
		return ir.OperatorGt
	default:
		return ir.OperatorGte
	}
}

func functionReturnsFreshValue(fn ir.Function) bool {
	foundReturn, allFresh := statementsReturnFreshValue(fn.Body, fn.IsConstructor)
	if !foundReturn {
		return true
	}
	return allFresh
}

func statementsReturnFreshValue(statements []ir.Statement, constructor bool) (bool, bool) {
	foundReturn := false
	for _, stmt := range statements {
		stmtFoundReturn, stmtAllFresh := statementReturnsFreshValue(stmt, constructor)
		if !stmtFoundReturn {
			continue
		}
		foundReturn = true
		if !stmtAllFresh {
			return true, false
		}
	}
	return foundReturn, true
}

func statementReturnsFreshValue(stmt ir.Statement, constructor bool) (bool, bool) {
	switch stmt := stmt.(type) {
	case *ir.ReturnStatement:
		return true, isFreshReturnExpression(stmt.Value, constructor)
	case *ir.IfStatement:
		foundA, freshA := statementsReturnFreshValue(stmt.Consequence, constructor)
		foundB, freshB := statementsReturnFreshValue(stmt.Alternative, constructor)
		return foundA || foundB, freshA && freshB
	case *ir.WhileStatement:
		return statementsReturnFreshValue(stmt.Body, constructor)
	case *ir.DoWhileStatement:
		return statementsReturnFreshValue(stmt.Body, constructor)
	case *ir.BlockStatement:
		return statementsReturnFreshValue(stmt.Body, constructor)
	case *ir.ForStatement:
		foundInit, freshInit := false, true
		if stmt.Init != nil {
			foundInit, freshInit = statementReturnsFreshValue(stmt.Init, constructor)
		}
		foundBody, freshBody := statementsReturnFreshValue(stmt.Body, constructor)
		foundUpdate, freshUpdate := false, true
		if stmt.Update != nil {
			foundUpdate, freshUpdate = statementReturnsFreshValue(stmt.Update, constructor)
		}
		return foundInit || foundBody || foundUpdate, freshInit && freshBody && freshUpdate
	case *ir.SwitchStatement:
		foundReturn := false
		for _, item := range stmt.Cases {
			caseFound, caseFresh := statementsReturnFreshValue(item.Consequent, constructor)
			if !caseFound {
				continue
			}
			foundReturn = true
			if !caseFresh {
				return true, false
			}
		}
		defaultFound, defaultFresh := statementsReturnFreshValue(stmt.Default, constructor)
		return foundReturn || defaultFound, defaultFresh
	case *ir.LabeledStatement:
		return statementReturnsFreshValue(stmt.Statement, constructor)
	case *ir.TryStatement:
		foundTry, freshTry := statementsReturnFreshValue(stmt.TryBody, constructor)
		foundCatch, freshCatch := statementsReturnFreshValue(stmt.CatchBody, constructor)
		foundFinally, freshFinally := statementsReturnFreshValue(stmt.FinallyBody, constructor)
		return foundTry || foundCatch || foundFinally, freshTry && freshCatch && freshFinally
	default:
		return false, true
	}
}

func isFreshReturnExpression(expr ir.Expression, constructor bool) bool {
	switch expr := expr.(type) {
	case *ir.NumberLiteral, *ir.BigIntLiteral, *ir.BooleanLiteral, *ir.NullLiteral, *ir.UndefinedLiteral, *ir.StringLiteral:
		return true
	case *ir.VariableRef:
		return constructor && expr.Name == "__self"
	case *ir.ObjectLiteral:
		for _, prop := range expr.Properties {
			if prop.Spread {
				return false
			}
			if !isFreshReturnExpression(prop.Value, constructor) {
				return false
			}
		}
		return true
	case *ir.ArrayLiteral:
		for _, item := range expr.Elements {
			if spread, ok := item.(*ir.SpreadExpression); ok {
				_ = spread
				return false
			}
			if !isFreshReturnExpression(item, constructor) {
				return false
			}
		}
		return true
	case *ir.TemplateLiteral:
		return true
	case *ir.FunctionValue:
		return true
	case *ir.CallExpression:
		if expr.Callee == "__jayess_constructor_return" && len(expr.Arguments) == 2 {
			return isFreshReturnExpression(expr.Arguments[1], constructor)
		}
		return false
	case *ir.UnaryExpression:
		return expr.Kind == ir.ValueNumber || expr.Kind == ir.ValueBoolean
	case *ir.TypeofExpression:
		return true
	case *ir.ComparisonExpression, *ir.InstanceofExpression:
		return true
	case *ir.BinaryExpression:
		return expr.Kind == ir.ValueNumber || expr.Kind == ir.ValueString || expr.Kind == ir.ValueBigInt || expr.Kind == ir.ValueBoolean
	default:
		return false
	}
}

func cloneKinds(input map[string]ir.ValueKind) map[string]ir.ValueKind {
	out := make(map[string]ir.ValueKind, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func nextLowerTemp(prefix string) string {
	id := atomic.AddUint64(&lowerTempCounter, 1)
	return fmt.Sprintf("__jayess_%s_%d", prefix, id)
}

func lowerForOfStatement(stmt *ast.ForOfStatement, symbols map[string]ir.ValueKind, functions map[string]bool) (ir.Statement, error) {
	itemsName := nextLowerTemp("items")
	indexName := nextLowerTemp("index")
	elementDecl := &ast.VariableDecl{Visibility: ast.VisibilityPublic, Kind: stmt.Kind, Name: stmt.Name, Value: &ast.IndexExpression{
		Target: &ast.Identifier{Name: itemsName},
		Index:  &ast.Identifier{Name: indexName},
	}}
	update := &ast.AssignmentStatement{
		Target: &ast.Identifier{Name: indexName},
		Value: &ast.BinaryExpression{
			Operator: ast.OperatorAdd,
			Left:     &ast.Identifier{Name: indexName},
			Right:    &ast.NumberLiteral{Value: 1},
		},
	}
	loop := &ast.ForStatement{
		Init: &ast.VariableDecl{Visibility: ast.VisibilityPublic, Kind: ast.DeclarationVar, Name: indexName, Value: &ast.NumberLiteral{Value: 0}},
		Condition: &ast.ComparisonExpression{
			Operator: ast.OperatorLt,
			Left:     &ast.Identifier{Name: indexName},
			Right:    &ast.MemberExpression{Target: &ast.Identifier{Name: itemsName}, Property: "length"},
		},
		Update: update,
		Body:   append([]ast.Statement{elementDecl}, stmt.Body...),
	}
	statements := []ast.Statement{
		&ast.VariableDecl{Visibility: ast.VisibilityPublic, Kind: ast.DeclarationVar, Name: itemsName, Value: &ast.CallExpression{Callee: "__jayess_iter_values", Arguments: []ast.Expression{stmt.Iterable}}},
		loop,
	}
	lowered, err := lowerStatements(statements, symbols, functions)
	if err != nil {
		return nil, err
	}
	return &ir.IfStatement{Condition: &ir.BooleanLiteral{Value: true}, Consequence: lowered}, nil
}

func lowerForInStatement(stmt *ast.ForInStatement, symbols map[string]ir.ValueKind, functions map[string]bool) (ir.Statement, error) {
	keysName := nextLowerTemp("keys")
	indexName := nextLowerTemp("index")
	keyDecl := &ast.VariableDecl{Visibility: ast.VisibilityPublic, Kind: stmt.Kind, Name: stmt.Name, Value: &ast.IndexExpression{
		Target: &ast.Identifier{Name: keysName},
		Index:  &ast.Identifier{Name: indexName},
	}}
	update := &ast.AssignmentStatement{
		Target: &ast.Identifier{Name: indexName},
		Value: &ast.BinaryExpression{
			Operator: ast.OperatorAdd,
			Left:     &ast.Identifier{Name: indexName},
			Right:    &ast.NumberLiteral{Value: 1},
		},
	}
	loop := &ast.ForStatement{
		Init: &ast.VariableDecl{Visibility: ast.VisibilityPublic, Kind: ast.DeclarationVar, Name: indexName, Value: &ast.NumberLiteral{Value: 0}},
		Condition: &ast.ComparisonExpression{
			Operator: ast.OperatorLt,
			Left:     &ast.Identifier{Name: indexName},
			Right:    &ast.MemberExpression{Target: &ast.Identifier{Name: keysName}, Property: "length"},
		},
		Update: update,
		Body:   append([]ast.Statement{keyDecl}, stmt.Body...),
	}
	statements := []ast.Statement{
		&ast.VariableDecl{Visibility: ast.VisibilityPublic, Kind: ast.DeclarationVar, Name: keysName, Value: &ast.CallExpression{Callee: "__jayess_object_keys", Arguments: []ast.Expression{stmt.Iterable}}},
		loop,
	}
	lowered, err := lowerStatements(statements, symbols, functions)
	if err != nil {
		return nil, err
	}
	return &ir.IfStatement{Condition: &ir.BooleanLiteral{Value: true}, Consequence: lowered}, nil
}

func lowerSwitchStatement(stmt *ast.SwitchStatement, symbols map[string]ir.ValueKind, functions map[string]bool) (ir.Statement, error) {
	discriminant, err := lowerExpression(stmt.Discriminant, symbols, functions)
	if err != nil {
		return nil, err
	}
	out := &ir.SwitchStatement{Discriminant: discriminant}
	for _, switchCase := range stmt.Cases {
		test, err := lowerExpression(switchCase.Test, symbols, functions)
		if err != nil {
			return nil, err
		}
		consequent, err := lowerStatements(switchCase.Consequent, symbols, functions)
		if err != nil {
			return nil, err
		}
		out.Cases = append(out.Cases, ir.SwitchCase{Test: test, Consequent: consequent})
	}
	defaultBody, err := lowerStatements(stmt.Default, symbols, functions)
	if err != nil {
		return nil, err
	}
	out.Default = defaultBody
	return out, nil
}
