package llvmbackend

import (
	"fmt"
	"math/big"

	"jayess-go/ast"
	jayessruntime "jayess-go/runtime"
)

func (emitter *ExpressionEmitter) emitComparisonExpression(expression *ast.ComparisonExpression) (string, error) {
	if result, ok, err := emitter.emitInExpression(expression); ok || err != nil {
		return result, err
	}
	if result, ok, err := emitter.emitStringComparisonExpression(expression); ok || err != nil {
		return result, err
	}
	if result, ok, err := emitter.emitBooleanComparisonExpression(expression); ok || err != nil {
		return result, err
	}
	if result, ok, err := emitter.emitNullishComparisonExpression(expression); ok || err != nil {
		return result, err
	}
	if result, ok, err := emitter.emitBigIntComparisonExpression(expression); ok || err != nil {
		return result, err
	}
	left, err := numericLiteralValue(expression.Left)
	if err != nil {
		return "", err
	}
	right, err := numericLiteralValue(expression.Right)
	if err != nil {
		return "", err
	}
	resultValue, err := evaluateNumericComparison(expression.Operator, left, right)
	if err != nil {
		return "", err
	}
	result := emitter.nextValueName()
	lowered, err := LowerRuntimeLiteral(result, RuntimeLiteral{Kind: jayessruntime.BooleanValue, Bool: resultValue}, emitter.stringIndex)
	if err != nil {
		return "", err
	}
	emitter.addDeclarations(lowered.Declarations)
	emitter.body = append(emitter.body, lowered.Body...)
	return result, nil
}

func (emitter *ExpressionEmitter) emitStringComparisonExpression(expression *ast.ComparisonExpression) (string, bool, error) {
	left, leftOK := expression.Left.(*ast.StringLiteral)
	right, rightOK := expression.Right.(*ast.StringLiteral)
	if !leftOK && !rightOK {
		return "", false, nil
	}
	if !leftOK || !rightOK {
		return "", true, fmt.Errorf("unsupported runtime string comparison operands %T and %T", expression.Left, expression.Right)
	}
	equal := left.Value == right.Value
	resultValue, err := evaluateStringEqualityComparison(expression.Operator, equal)
	if err != nil {
		return "", true, err
	}
	result := emitter.nextValueName()
	lowered, err := LowerRuntimeLiteral(result, RuntimeLiteral{Kind: jayessruntime.BooleanValue, Bool: resultValue}, emitter.stringIndex)
	if err != nil {
		return "", true, err
	}
	emitter.addDeclarations(lowered.Declarations)
	emitter.body = append(emitter.body, lowered.Body...)
	return result, true, nil
}

func (emitter *ExpressionEmitter) emitBooleanComparisonExpression(expression *ast.ComparisonExpression) (string, bool, error) {
	left, leftOK := expression.Left.(*ast.BooleanLiteral)
	right, rightOK := expression.Right.(*ast.BooleanLiteral)
	if !leftOK && !rightOK {
		return "", false, nil
	}
	if !leftOK || !rightOK {
		return "", true, fmt.Errorf("unsupported runtime boolean comparison operands %T and %T", expression.Left, expression.Right)
	}
	equal := left.Value == right.Value
	resultValue, err := evaluatePrimitiveEqualityComparison(expression.Operator, equal, "boolean")
	if err != nil {
		return "", true, err
	}
	result := emitter.nextValueName()
	lowered, err := LowerRuntimeLiteral(result, RuntimeLiteral{Kind: jayessruntime.BooleanValue, Bool: resultValue}, emitter.stringIndex)
	if err != nil {
		return "", true, err
	}
	emitter.addDeclarations(lowered.Declarations)
	emitter.body = append(emitter.body, lowered.Body...)
	return result, true, nil
}

func (emitter *ExpressionEmitter) emitNullishComparisonExpression(expression *ast.ComparisonExpression) (string, bool, error) {
	leftKind, leftOK := nullishLiteralKind(expression.Left)
	rightKind, rightOK := nullishLiteralKind(expression.Right)
	if !leftOK && !rightOK {
		return "", false, nil
	}
	if !leftOK || !rightOK {
		return "", true, fmt.Errorf("unsupported runtime nullish comparison operands %T and %T", expression.Left, expression.Right)
	}
	equal := leftKind == rightKind
	if expression.Operator == ast.OperatorEq || expression.Operator == ast.OperatorNe {
		equal = true
	}
	resultValue, err := evaluatePrimitiveEqualityComparison(expression.Operator, equal, "nullish")
	if err != nil {
		return "", true, err
	}
	result := emitter.nextValueName()
	lowered, err := LowerRuntimeLiteral(result, RuntimeLiteral{Kind: jayessruntime.BooleanValue, Bool: resultValue}, emitter.stringIndex)
	if err != nil {
		return "", true, err
	}
	emitter.addDeclarations(lowered.Declarations)
	emitter.body = append(emitter.body, lowered.Body...)
	return result, true, nil
}

func (emitter *ExpressionEmitter) emitBigIntComparisonExpression(expression *ast.ComparisonExpression) (string, bool, error) {
	leftLiteral, leftOK := expression.Left.(*ast.BigIntLiteral)
	rightLiteral, rightOK := expression.Right.(*ast.BigIntLiteral)
	if !leftOK && !rightOK {
		return "", false, nil
	}
	if !leftOK || !rightOK {
		return "", true, fmt.Errorf("unsupported runtime bigint comparison operands %T and %T", expression.Left, expression.Right)
	}
	left, ok := new(big.Int).SetString(leftLiteral.Value, 10)
	if !ok {
		return "", true, fmt.Errorf("invalid bigint literal %q", leftLiteral.Value)
	}
	right, ok := new(big.Int).SetString(rightLiteral.Value, 10)
	if !ok {
		return "", true, fmt.Errorf("invalid bigint literal %q", rightLiteral.Value)
	}
	resultValue, err := evaluateBigIntComparison(expression.Operator, left.Cmp(right))
	if err != nil {
		return "", true, err
	}
	result := emitter.nextValueName()
	lowered, err := LowerRuntimeLiteral(result, RuntimeLiteral{Kind: jayessruntime.BooleanValue, Bool: resultValue}, emitter.stringIndex)
	if err != nil {
		return "", true, err
	}
	emitter.addDeclarations(lowered.Declarations)
	emitter.body = append(emitter.body, lowered.Body...)
	return result, true, nil
}

func nullishLiteralKind(expression ast.Expression) (string, bool) {
	switch expression.(type) {
	case *ast.NullLiteral:
		return "null", true
	case *ast.UndefinedLiteral:
		return "undefined", true
	default:
		return "", false
	}
}

func evaluateStringEqualityComparison(operator ast.ComparisonOperator, equal bool) (bool, error) {
	return evaluatePrimitiveEqualityComparison(operator, equal, "string")
}

func evaluatePrimitiveEqualityComparison(operator ast.ComparisonOperator, equal bool, kind string) (bool, error) {
	evaluate, ok := equalityComparisonDispatchers()[operator]
	if !ok {
		return false, fmt.Errorf("unsupported runtime %s comparison operator %s", kind, operator)
	}
	return evaluate(equal), nil
}

func evaluateBigIntComparison(operator ast.ComparisonOperator, comparison int) (bool, error) {
	evaluate, ok := orderedComparisonDispatchers()[operator]
	if !ok {
		return false, fmt.Errorf("unsupported runtime bigint comparison operator %s", operator)
	}
	return evaluate(comparison), nil
}

func evaluateNumericComparison(operator ast.ComparisonOperator, left float64, right float64) (bool, error) {
	evaluate, ok := orderedComparisonDispatchers()[operator]
	if !ok {
		return false, fmt.Errorf("unsupported runtime numeric comparison operator %s", operator)
	}
	return evaluate(compareFloat64(left, right)), nil
}

type equalityComparisonEvaluator func(bool) bool
type orderedComparisonEvaluator func(int) bool

func equalityComparisonDispatchers() map[ast.ComparisonOperator]equalityComparisonEvaluator {
	return map[ast.ComparisonOperator]equalityComparisonEvaluator{
		ast.OperatorEq:       func(equal bool) bool { return equal },
		ast.OperatorStrictEq: func(equal bool) bool { return equal },
		ast.OperatorNe:       func(equal bool) bool { return !equal },
		ast.OperatorStrictNe: func(equal bool) bool { return !equal },
	}
}

func orderedComparisonDispatchers() map[ast.ComparisonOperator]orderedComparisonEvaluator {
	return map[ast.ComparisonOperator]orderedComparisonEvaluator{
		ast.OperatorEq:       func(comparison int) bool { return comparison == 0 },
		ast.OperatorStrictEq: func(comparison int) bool { return comparison == 0 },
		ast.OperatorNe:       func(comparison int) bool { return comparison != 0 },
		ast.OperatorStrictNe: func(comparison int) bool { return comparison != 0 },
		ast.OperatorLt:       func(comparison int) bool { return comparison < 0 },
		ast.OperatorLte:      func(comparison int) bool { return comparison <= 0 },
		ast.OperatorGt:       func(comparison int) bool { return comparison > 0 },
		ast.OperatorGte:      func(comparison int) bool { return comparison >= 0 },
	}
}

func compareFloat64(left float64, right float64) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}
