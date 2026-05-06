package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
	jayessruntime "jayess-go/runtime"
)

const (
	runtimeClassConstructSymbol      = "jayess_class_construct"
	runtimeClassConstructSuperSymbol = "jayess_class_construct_super"
	runtimeCurrentThisSymbol         = "jayess_current_this"
	runtimeNewTargetSymbol           = "jayess_new_target"
)

func (emitter *ExpressionEmitter) emitNewExpression(expression *ast.NewExpression) (string, error) {
	if expression == nil {
		return "", fmt.Errorf("new expression must not be nil")
	}
	callee, err := emitter.EmitExpression(expression.Callee)
	if err != nil {
		return "", err
	}
	arguments, err := emitter.emitCallArguments(expression.Arguments)
	if err != nil {
		return "", err
	}
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: callee},
		{IRType: runtimeValueIRType, Value: arguments},
	}
	return emitter.emitRuntimeValueCall(runtimeClassConstructSymbol, args)
}

func (emitter *ExpressionEmitter) emitThisExpression(expression *ast.ThisExpression) (string, error) {
	if expression == nil {
		return "", fmt.Errorf("this expression must not be nil")
	}
	return emitter.emitRuntimeValueCall(runtimeCurrentThisSymbol, nil)
}

func (emitter *ExpressionEmitter) emitSuperExpression(expression *ast.SuperExpression) (string, error) {
	if expression == nil {
		return "", fmt.Errorf("super expression must not be nil")
	}
	return emitter.emitRuntimeValueCall(runtimeNewTargetSymbol, nil)
}

func (emitter *ExpressionEmitter) emitNewTargetExpression(expression *ast.NewTargetExpression) (string, error) {
	if expression == nil {
		return "", fmt.Errorf("new.target expression must not be nil")
	}
	return emitter.emitRuntimeValueCall(runtimeNewTargetSymbol, nil)
}

func (emitter *ExpressionEmitter) emitRuntimeValueCall(symbol string, args []RuntimeCallArg) (string, error) {
	result := emitter.nextValueName()
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(symbol, runtimeValueIRType, args)})
	emitter.body = append(emitter.body, RuntimeCall(result, runtimeValueIRType, symbol, args))
	return result, nil
}

func (emitter *ExpressionEmitter) emitUndefinedThisValue() (string, error) {
	return emitter.EmitRuntimeLiteral(RuntimeLiteral{Kind: jayessruntime.UndefinedValue})
}
