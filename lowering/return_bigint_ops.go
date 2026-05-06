package lowering

import (
	"math/big"

	"jayess-go/ast"
)

func evaluateBigIntBinary(operator ast.BinaryOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (string, bool) {
	left, ok := evaluateBigIntValue(leftExpression, bindings)
	if !ok {
		return "", false
	}
	right, ok := evaluateBigIntValue(rightExpression, bindings)
	if !ok {
		return "", false
	}
	value, ok := evaluateBigIntBinaryValue(operator, left, right)
	if !ok {
		return "", false
	}
	return value, true
}

func evaluateBigIntBinaryValue(operator ast.BinaryOperator, leftValue string, rightValue string) (string, bool) {
	left, ok := parseBigIntValue(leftValue)
	if !ok {
		return "", false
	}
	right, ok := parseBigIntValue(rightValue)
	if !ok {
		return "", false
	}
	result := new(big.Int)
	switch operator {
	case ast.OperatorAdd:
		result.Add(left, right)
	case ast.OperatorSub:
		result.Sub(left, right)
	case ast.OperatorMul:
		result.Mul(left, right)
	case ast.OperatorDiv:
		if right.Sign() == 0 {
			return "", false
		}
		result.Quo(left, right)
	case ast.OperatorMod:
		if right.Sign() == 0 {
			return "", false
		}
		result.Rem(left, right)
	case ast.OperatorPow:
		if right.Sign() < 0 || !right.IsInt64() || right.Int64() > 64 {
			return "", false
		}
		result.Exp(left, right, nil)
	case ast.OperatorBitAnd:
		result.And(left, right)
	case ast.OperatorBitOr:
		result.Or(left, right)
	case ast.OperatorBitXor:
		result.Xor(left, right)
	case ast.OperatorShl:
		count, ok := bigIntShiftCount(right)
		if !ok {
			return "", false
		}
		result.Lsh(left, count)
	case ast.OperatorShr:
		count, ok := bigIntShiftCount(right)
		if !ok {
			return "", false
		}
		result.Rsh(left, count)
	default:
		return "", false
	}
	return result.String(), true
}

func bigIntShiftCount(value *big.Int) (uint, bool) {
	if value.Sign() < 0 || !value.IsUint64() || value.Uint64() > 1024 {
		return 0, false
	}
	return uint(value.Uint64()), true
}

func assignmentBigIntBinaryOperator(operator ast.AssignmentOperator) (ast.BinaryOperator, bool) {
	switch operator {
	case ast.AssignmentAddAssign:
		return ast.OperatorAdd, true
	case ast.AssignmentSubAssign:
		return ast.OperatorSub, true
	case ast.AssignmentMulAssign:
		return ast.OperatorMul, true
	case ast.AssignmentDivAssign:
		return ast.OperatorDiv, true
	case ast.AssignmentModAssign:
		return ast.OperatorMod, true
	case ast.AssignmentPowAssign:
		return ast.OperatorPow, true
	case ast.AssignmentBitAndAssign:
		return ast.OperatorBitAnd, true
	case ast.AssignmentBitOrAssign:
		return ast.OperatorBitOr, true
	case ast.AssignmentBitXorAssign:
		return ast.OperatorBitXor, true
	case ast.AssignmentShlAssign:
		return ast.OperatorShl, true
	case ast.AssignmentShrAssign:
		return ast.OperatorShr, true
	default:
		return "", false
	}
}
