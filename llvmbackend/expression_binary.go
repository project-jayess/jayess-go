package llvmbackend

import (
	"fmt"
	"math"

	"jayess-go/ast"
	jayessruntime "jayess-go/runtime"
)

func (emitter *ExpressionEmitter) emitBinaryExpression(expression *ast.BinaryExpression) (string, error) {
	if expression.Operator == ast.OperatorAdd {
		if result, ok, err := emitter.emitStringConcatExpression(expression); ok || err != nil {
			return result, err
		}
	}
	if result, ok, err := emitter.emitBigIntBinaryExpression(expression); ok || err != nil {
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
	resultValue, err := evaluateNumericBinary(expression.Operator, left, right)
	if err != nil {
		return "", err
	}
	result := emitter.nextValueName()
	lowered, err := LowerRuntimeLiteral(result, RuntimeLiteral{Kind: jayessruntime.NumberValue, Number: resultValue}, emitter.stringIndex)
	if err != nil {
		return "", err
	}
	emitter.addDeclarations(lowered.Declarations)
	emitter.body = append(emitter.body, lowered.Body...)
	return result, nil
}

func (emitter *ExpressionEmitter) emitStringConcatExpression(expression *ast.BinaryExpression) (string, bool, error) {
	left, leftOK := expression.Left.(*ast.StringLiteral)
	right, rightOK := expression.Right.(*ast.StringLiteral)
	if !leftOK && !rightOK {
		return "", false, nil
	}
	if !leftOK || !rightOK {
		return "", true, fmt.Errorf("unsupported runtime string concatenation operands %T and %T", expression.Left, expression.Right)
	}
	result := emitter.nextValueName()
	lowered, err := LowerRuntimeLiteral(result, RuntimeLiteral{Kind: jayessruntime.StringValue, Text: left.Value + right.Value}, emitter.stringIndex)
	if err != nil {
		return "", true, err
	}
	emitter.addDeclarations(lowered.Declarations)
	emitter.globals = append(emitter.globals, lowered.Globals...)
	emitter.stringIndex += len(lowered.Globals)
	emitter.body = append(emitter.body, lowered.Body...)
	return result, true, nil
}

func numericLiteralValue(expression ast.Expression) (float64, error) {
	literal, ok := expression.(*ast.NumberLiteral)
	if !ok {
		return 0, fmt.Errorf("unsupported runtime numeric binary operand %T", expression)
	}
	value, err := RuntimeLiteralFromAST(literal)
	if err != nil {
		return 0, err
	}
	return value.Number, nil
}

func evaluateNumericBinary(operator ast.BinaryOperator, left float64, right float64) (float64, error) {
	evaluate, ok := numericBinaryDispatchers()[operator]
	if !ok {
		return 0, fmt.Errorf("unsupported runtime numeric binary operator %s", operator)
	}
	return evaluate(left, right), nil
}

type numericBinaryEvaluator func(float64, float64) float64

func numericBinaryDispatchers() map[ast.BinaryOperator]numericBinaryEvaluator {
	return map[ast.BinaryOperator]numericBinaryEvaluator{
		ast.OperatorAdd: func(left float64, right float64) float64 {
			return left + right
		},
		ast.OperatorSub: func(left float64, right float64) float64 {
			return left - right
		},
		ast.OperatorMul: func(left float64, right float64) float64 {
			return left * right
		},
		ast.OperatorDiv: func(left float64, right float64) float64 {
			return left / right
		},
		ast.OperatorMod: math.Mod,
		ast.OperatorPow: math.Pow,
	}
}
