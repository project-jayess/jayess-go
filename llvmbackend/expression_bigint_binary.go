package llvmbackend

import (
	"fmt"
	"math/big"

	"jayess-go/ast"
)

func (emitter *ExpressionEmitter) emitBigIntBinaryExpression(expression *ast.BinaryExpression) (string, bool, error) {
	leftLiteral, leftOK := expression.Left.(*ast.BigIntLiteral)
	rightLiteral, rightOK := expression.Right.(*ast.BigIntLiteral)
	if !leftOK && !rightOK {
		return "", false, nil
	}
	if !leftOK || !rightOK {
		return "", true, fmt.Errorf("unsupported runtime bigint binary operands %T and %T", expression.Left, expression.Right)
	}
	left, ok := parseBigIntLiteral(leftLiteral)
	if !ok {
		return "", true, fmt.Errorf("invalid bigint literal %q", leftLiteral.Value)
	}
	right, ok := parseBigIntLiteral(rightLiteral)
	if !ok {
		return "", true, fmt.Errorf("invalid bigint literal %q", rightLiteral.Value)
	}
	result, err := evaluateBigIntBinary(expression.Operator, left, right)
	if err != nil {
		return "", true, err
	}
	value, err := emitter.emitBigIntLiteralText(result.String())
	return value, true, err
}

func evaluateBigIntBinary(operator ast.BinaryOperator, left *big.Int, right *big.Int) (*big.Int, error) {
	evaluate, ok := bigIntBinaryDispatchers()[operator]
	if !ok {
		return nil, fmt.Errorf("unsupported runtime bigint binary operator %s", operator)
	}
	return evaluate(left, right)
}

type bigIntBinaryEvaluator func(*big.Int, *big.Int) (*big.Int, error)

func bigIntBinaryDispatchers() map[ast.BinaryOperator]bigIntBinaryEvaluator {
	return map[ast.BinaryOperator]bigIntBinaryEvaluator{
		ast.OperatorAdd:    evaluateBigIntAdd,
		ast.OperatorSub:    evaluateBigIntSub,
		ast.OperatorMul:    evaluateBigIntMul,
		ast.OperatorDiv:    evaluateBigIntDiv,
		ast.OperatorMod:    evaluateBigIntMod,
		ast.OperatorPow:    evaluateBigIntPow,
		ast.OperatorBitAnd: evaluateBigIntBitAnd,
		ast.OperatorBitOr:  evaluateBigIntBitOr,
		ast.OperatorBitXor: evaluateBigIntBitXor,
		ast.OperatorShl:    evaluateBigIntShl,
		ast.OperatorShr:    evaluateBigIntShr,
	}
}

func evaluateBigIntAdd(left *big.Int, right *big.Int) (*big.Int, error) {
	return new(big.Int).Add(left, right), nil
}

func evaluateBigIntSub(left *big.Int, right *big.Int) (*big.Int, error) {
	return new(big.Int).Sub(left, right), nil
}

func evaluateBigIntMul(left *big.Int, right *big.Int) (*big.Int, error) {
	return new(big.Int).Mul(left, right), nil
}

func evaluateBigIntDiv(left *big.Int, right *big.Int) (*big.Int, error) {
	if right.Sign() == 0 {
		return nil, fmt.Errorf("unsupported runtime bigint division by zero")
	}
	return new(big.Int).Quo(left, right), nil
}

func evaluateBigIntMod(left *big.Int, right *big.Int) (*big.Int, error) {
	if right.Sign() == 0 {
		return nil, fmt.Errorf("unsupported runtime bigint remainder by zero")
	}
	return new(big.Int).Rem(left, right), nil
}

func evaluateBigIntPow(left *big.Int, right *big.Int) (*big.Int, error) {
	if right.Sign() < 0 || !right.IsInt64() || right.Int64() > 64 {
		return nil, fmt.Errorf("unsupported runtime bigint exponent %s", right.String())
	}
	return new(big.Int).Exp(left, right, nil), nil
}

func evaluateBigIntBitAnd(left *big.Int, right *big.Int) (*big.Int, error) {
	return new(big.Int).And(left, right), nil
}

func evaluateBigIntBitOr(left *big.Int, right *big.Int) (*big.Int, error) {
	return new(big.Int).Or(left, right), nil
}

func evaluateBigIntBitXor(left *big.Int, right *big.Int) (*big.Int, error) {
	return new(big.Int).Xor(left, right), nil
}

func evaluateBigIntShl(left *big.Int, right *big.Int) (*big.Int, error) {
	count, ok := bigIntShiftCount(right)
	if !ok {
		return nil, fmt.Errorf("unsupported runtime bigint left shift count %s", right.String())
	}
	return new(big.Int).Lsh(left, count), nil
}

func evaluateBigIntShr(left *big.Int, right *big.Int) (*big.Int, error) {
	count, ok := bigIntShiftCount(right)
	if !ok {
		return nil, fmt.Errorf("unsupported runtime bigint right shift count %s", right.String())
	}
	return new(big.Int).Rsh(left, count), nil
}

func bigIntShiftCount(value *big.Int) (uint, bool) {
	if value.Sign() < 0 || !value.IsUint64() || value.Uint64() > 1024 {
		return 0, false
	}
	return uint(value.Uint64()), true
}
