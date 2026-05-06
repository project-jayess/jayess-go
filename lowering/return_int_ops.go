package lowering

import (
	"math"

	"jayess-go/ast"
)

func evaluateIntAssignment(operator ast.AssignmentOperator, left int, right int) (int, bool) {
	switch operator {
	case ast.AssignmentAddAssign:
		return left + right, true
	case ast.AssignmentSubAssign:
		return left - right, true
	case ast.AssignmentMulAssign:
		return left * right, true
	case ast.AssignmentPowAssign:
		return int(math.Pow(float64(left), float64(right))), true
	case ast.AssignmentDivAssign:
		if right == 0 {
			return 0, false
		}
		return left / right, true
	case ast.AssignmentModAssign:
		if right == 0 {
			return 0, false
		}
		return left % right, true
	case ast.AssignmentBitAndAssign:
		return left & right, true
	case ast.AssignmentBitOrAssign:
		return left | right, true
	case ast.AssignmentBitXorAssign:
		return left ^ right, true
	case ast.AssignmentShlAssign:
		return left << shiftCount(right), true
	case ast.AssignmentShrAssign:
		return left >> shiftCount(right), true
	case ast.AssignmentUShrAssign:
		return int(uint32(left) >> shiftCount(right)), true
	default:
		return 0, false
	}
}

func evaluateIntBinary(operator ast.BinaryOperator, left int, right int) (int, bool) {
	switch operator {
	case ast.OperatorAdd:
		return left + right, true
	case ast.OperatorSub:
		return left - right, true
	case ast.OperatorMul:
		return left * right, true
	case ast.OperatorDiv:
		if right == 0 {
			return 0, false
		}
		return left / right, true
	case ast.OperatorMod:
		if right == 0 {
			return 0, false
		}
		return left % right, true
	case ast.OperatorPow:
		return int(math.Pow(float64(left), float64(right))), true
	case ast.OperatorBitAnd:
		return left & right, true
	case ast.OperatorBitOr:
		return left | right, true
	case ast.OperatorBitXor:
		return left ^ right, true
	case ast.OperatorShl:
		return left << shiftCount(right), true
	case ast.OperatorShr:
		return left >> shiftCount(right), true
	case ast.OperatorUShr:
		return int(uint32(left) >> shiftCount(right)), true
	default:
		return 0, false
	}
}

func shiftCount(value int) uint {
	return uint(value) & 31
}
