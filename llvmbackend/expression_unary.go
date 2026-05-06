package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
	jayessruntime "jayess-go/runtime"
)

type unaryExpressionEmitter func(*ExpressionEmitter, *ast.UnaryExpression) (string, error)

func (emitter *ExpressionEmitter) emitUnaryExpression(expression *ast.UnaryExpression) (string, error) {
	lower, ok := map[ast.UnaryOperator]unaryExpressionEmitter{
		ast.OperatorVoid:     (*ExpressionEmitter).emitVoidExpression,
		ast.OperatorDelete:   (*ExpressionEmitter).emitDeleteExpression,
		ast.OperatorBitNot:   (*ExpressionEmitter).emitBigIntBitNotExpression,
		ast.OperatorNegate:   (*ExpressionEmitter).emitNegateExpression,
		ast.OperatorPositive: (*ExpressionEmitter).emitNumericUnaryExpression,
		ast.OperatorNot:      (*ExpressionEmitter).emitBooleanNotExpression,
	}[expression.Operator]
	if !ok {
		return "", fmt.Errorf("unsupported runtime unary operator %s", expression.Operator)
	}
	return lower(emitter, expression)
}

func (emitter *ExpressionEmitter) emitNegateExpression(expression *ast.UnaryExpression) (string, error) {
	if _, ok := expression.Right.(*ast.BigIntLiteral); ok {
		return emitter.emitBigIntNegateExpression(expression)
	}
	return emitter.emitNumericUnaryExpression(expression)
}

func (emitter *ExpressionEmitter) emitVoidExpression(expression *ast.UnaryExpression) (string, error) {
	if _, err := emitter.EmitExpressionSequence(expression.Right); err != nil {
		return "", err
	}
	result := emitter.nextValueName()
	lowered, err := LowerRuntimeLiteral(result, RuntimeLiteral{Kind: jayessruntime.UndefinedValue}, emitter.stringIndex)
	if err != nil {
		return "", err
	}
	emitter.addDeclarations(lowered.Declarations)
	emitter.body = append(emitter.body, lowered.Body...)
	return result, nil
}

func (emitter *ExpressionEmitter) emitBooleanNotExpression(expression *ast.UnaryExpression) (string, error) {
	literal, ok := expression.Right.(*ast.BooleanLiteral)
	if !ok {
		return "", fmt.Errorf("unsupported runtime boolean-not operand %T", expression.Right)
	}
	result := emitter.nextValueName()
	lowered, err := LowerRuntimeLiteral(result, RuntimeLiteral{Kind: jayessruntime.BooleanValue, Bool: !literal.Value}, emitter.stringIndex)
	if err != nil {
		return "", err
	}
	emitter.addDeclarations(lowered.Declarations)
	emitter.body = append(emitter.body, lowered.Body...)
	return result, nil
}

func (emitter *ExpressionEmitter) emitNumericUnaryExpression(expression *ast.UnaryExpression) (string, error) {
	literal, ok := expression.Right.(*ast.NumberLiteral)
	if !ok {
		return "", fmt.Errorf("unsupported runtime numeric unary operand %T", expression.Right)
	}
	value, err := RuntimeLiteralFromAST(literal)
	if err != nil {
		return "", err
	}
	if expression.Operator == ast.OperatorNegate {
		value.Number = -value.Number
	}
	result := emitter.nextValueName()
	lowered, err := LowerRuntimeLiteral(result, value, emitter.stringIndex)
	if err != nil {
		return "", err
	}
	emitter.addDeclarations(lowered.Declarations)
	emitter.body = append(emitter.body, lowered.Body...)
	return result, nil
}
