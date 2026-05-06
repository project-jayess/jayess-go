package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
)

const (
	runtimeUpdateIncrementSymbol = "jayess_value_update_increment"
	runtimeUpdateDecrementSymbol = "jayess_value_update_decrement"
)

func (emitter *ExpressionEmitter) emitUpdateExpression(expression *ast.UpdateExpression) (string, error) {
	target, err := emitter.ResolveAssignmentTarget(expression.Target)
	if err != nil {
		return "", err
	}
	previous, err := target.Load()
	if err != nil {
		return "", err
	}
	next, err := emitter.emitUpdateValue(expression.Operator, previous)
	if err != nil {
		return "", err
	}
	if err := target.Store(next); err != nil {
		return "", err
	}
	if expression.Prefix {
		return next, nil
	}
	return previous, nil
}

func (emitter *ExpressionEmitter) emitUpdateValue(operator ast.UpdateOperator, value string) (string, error) {
	symbol, ok := map[ast.UpdateOperator]string{
		ast.UpdateIncrement: runtimeUpdateIncrementSymbol,
		ast.UpdateDecrement: runtimeUpdateDecrementSymbol,
	}[operator]
	if !ok {
		return "", fmt.Errorf("unsupported runtime update operator %s", operator)
	}
	args := []RuntimeCallArg{{IRType: runtimeValueIRType, Value: value}}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(symbol, runtimeValueIRType, args)})
	result := emitter.nextValueName()
	emitter.body = append(emitter.body, RuntimeCall(result, runtimeValueIRType, symbol, args))
	return result, nil
}
