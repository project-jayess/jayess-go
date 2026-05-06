package llvmbackend

import (
	"fmt"
	"math/big"

	"jayess-go/ast"
	jayessruntime "jayess-go/runtime"
)

func (emitter *ExpressionEmitter) emitBigIntNegateExpression(expression *ast.UnaryExpression) (string, error) {
	literal, ok := expression.Right.(*ast.BigIntLiteral)
	if !ok {
		return "", fmt.Errorf("unsupported runtime bigint negate operand %T", expression.Right)
	}
	value, ok := parseBigIntLiteral(literal)
	if !ok {
		return "", fmt.Errorf("invalid bigint literal %q", literal.Value)
	}
	value.Neg(value)
	return emitter.emitBigIntLiteralText(value.String())
}

func (emitter *ExpressionEmitter) emitBigIntBitNotExpression(expression *ast.UnaryExpression) (string, error) {
	literal, ok := expression.Right.(*ast.BigIntLiteral)
	if !ok {
		return "", fmt.Errorf("unsupported runtime bigint bitwise-not operand %T", expression.Right)
	}
	value, ok := parseBigIntLiteral(literal)
	if !ok {
		return "", fmt.Errorf("invalid bigint literal %q", literal.Value)
	}
	value.Not(value)
	return emitter.emitBigIntLiteralText(value.String())
}

func (emitter *ExpressionEmitter) emitBigIntLiteralText(text string) (string, error) {
	result := emitter.nextValueName()
	lowered, err := LowerRuntimeLiteral(result, RuntimeLiteral{Kind: jayessruntime.BigIntValue, Text: text}, emitter.stringIndex)
	if err != nil {
		return "", err
	}
	emitter.addDeclarations(lowered.Declarations)
	emitter.globals = append(emitter.globals, lowered.Globals...)
	emitter.stringIndex += len(lowered.Globals)
	emitter.body = append(emitter.body, lowered.Body...)
	return result, nil
}

func parseBigIntLiteral(literal *ast.BigIntLiteral) (*big.Int, bool) {
	return new(big.Int).SetString(literal.Value, 10)
}
