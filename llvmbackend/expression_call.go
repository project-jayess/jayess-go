package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
	jayessruntime "jayess-go/runtime"
)

const (
	runtimeApplyFunctionSymbol = "jayess_apply_function"
	runtimeBindFunctionSymbol  = "jayess_bind_function"
	runtimeCallFunctionSymbol  = "jayess_call_function"
)

func (emitter *ExpressionEmitter) emitCallExpression(expression *ast.CallExpression) (string, error) {
	if expression == nil {
		return "", fmt.Errorf("call expression must not be nil")
	}
	callee, err := emitter.LoadLocal(expression.Callee)
	if err != nil {
		return "", err
	}
	thisValue, err := emitter.EmitRuntimeLiteral(RuntimeLiteral{Kind: jayessruntime.UndefinedValue})
	if err != nil {
		return "", err
	}
	arguments, err := emitter.emitCallArguments(expression.Arguments)
	if err != nil {
		return "", err
	}
	return emitter.emitRuntimeCallable(runtimeCallFunctionSymbol, callee, thisValue, arguments)
}

func (emitter *ExpressionEmitter) emitInvokeExpression(expression *ast.InvokeExpression) (string, error) {
	if expression == nil {
		return "", fmt.Errorf("invoke expression must not be nil")
	}
	if expression.Optional {
		return "", fmt.Errorf("unsupported runtime optional call lowering")
	}
	if _, ok := expression.Callee.(*ast.SuperExpression); ok {
		return emitter.emitSuperConstructorCall(expression.Arguments)
	}
	if member, ok := expression.Callee.(*ast.MemberExpression); ok {
		return emitter.emitMemberInvoke(member, expression.Arguments)
	}
	callee, err := emitter.EmitExpression(expression.Callee)
	if err != nil {
		return "", err
	}
	thisValue, err := emitter.EmitRuntimeLiteral(RuntimeLiteral{Kind: jayessruntime.UndefinedValue})
	if err != nil {
		return "", err
	}
	arguments, err := emitter.emitCallArguments(expression.Arguments)
	if err != nil {
		return "", err
	}
	return emitter.emitRuntimeCallable(runtimeCallFunctionSymbol, callee, thisValue, arguments)
}

func (emitter *ExpressionEmitter) emitSuperConstructorCall(arguments []ast.Expression) (string, error) {
	thisValue, err := emitter.emitRuntimeValueCall(runtimeCurrentThisSymbol, nil)
	if err != nil {
		return "", err
	}
	argsValue, err := emitter.emitCallArguments(arguments)
	if err != nil {
		return "", err
	}
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: thisValue},
		{IRType: runtimeValueIRType, Value: argsValue},
	}
	return emitter.emitRuntimeValueCall(runtimeClassConstructSuperSymbol, args)
}

func (emitter *ExpressionEmitter) emitMemberInvoke(member *ast.MemberExpression, arguments []ast.Expression) (string, error) {
	if member.Optional {
		return "", fmt.Errorf("unsupported runtime optional member call lowering")
	}
	if member.Private {
		return "", fmt.Errorf("unsupported runtime private member call lowering")
	}
	switch member.Property {
	case "bind":
		return emitter.emitCallableHelperInvoke(runtimeBindFunctionSymbol, member.Target, arguments)
	case "call":
		return emitter.emitCallableHelperInvoke(runtimeCallFunctionSymbol, member.Target, arguments)
	case "apply":
		return emitter.emitApplyInvoke(member.Target, arguments)
	default:
		return emitter.emitMethodInvoke(member, arguments)
	}
}

func (emitter *ExpressionEmitter) emitCallableHelperInvoke(symbol string, target ast.Expression, arguments []ast.Expression) (string, error) {
	callee, err := emitter.EmitExpression(target)
	if err != nil {
		return "", err
	}
	thisValue, rest, err := emitter.emitThisAndRestArguments(arguments)
	if err != nil {
		return "", err
	}
	return emitter.emitRuntimeCallable(symbol, callee, thisValue, rest)
}

func (emitter *ExpressionEmitter) emitApplyInvoke(target ast.Expression, arguments []ast.Expression) (string, error) {
	callee, err := emitter.EmitExpression(target)
	if err != nil {
		return "", err
	}
	thisValue, rest, err := emitter.emitThisAndRestArguments(arguments)
	if err != nil {
		return "", err
	}
	if len(arguments) > 1 {
		applied, err := emitter.EmitExpression(arguments[1])
		if err != nil {
			return "", err
		}
		rest = applied
	}
	return emitter.emitRuntimeCallable(runtimeApplyFunctionSymbol, callee, thisValue, rest)
}

func (emitter *ExpressionEmitter) emitMethodInvoke(member *ast.MemberExpression, arguments []ast.Expression) (string, error) {
	object, err := emitter.EmitExpression(member.Target)
	if err != nil {
		return "", err
	}
	key, err := emitter.EmitRuntimeLiteral(RuntimeLiteral{Kind: jayessruntime.StringValue, Text: member.Property})
	if err != nil {
		return "", err
	}
	callee := runtimeSetAssignmentTarget{
		emitter: emitter,
		getter:  runtimeGetPropertySymbol,
		setter:  runtimeSetPropertySymbol,
		deleter: runtimeDeletePropertySymbol,
		object:  object,
		key:     key,
	}
	function, err := callee.Load()
	if err != nil {
		return "", err
	}
	args, err := emitter.emitCallArguments(arguments)
	if err != nil {
		return "", err
	}
	return emitter.emitRuntimeCallable(runtimeCallFunctionSymbol, function, object, args)
}

func (emitter *ExpressionEmitter) emitThisAndRestArguments(arguments []ast.Expression) (string, string, error) {
	if len(arguments) == 0 {
		thisValue, err := emitter.EmitRuntimeLiteral(RuntimeLiteral{Kind: jayessruntime.UndefinedValue})
		if err != nil {
			return "", "", err
		}
		rest, err := emitter.emitCallArguments(nil)
		return thisValue, rest, err
	}
	thisValue, err := emitter.EmitExpression(arguments[0])
	if err != nil {
		return "", "", err
	}
	rest, err := emitter.emitCallArguments(arguments[1:])
	return thisValue, rest, err
}

func (emitter *ExpressionEmitter) emitCallArguments(arguments []ast.Expression) (string, error) {
	array, err := emitter.emitRuntimeArrayNew()
	if err != nil {
		return "", err
	}
	for _, argument := range arguments {
		if spread, ok := argument.(*ast.SpreadExpression); ok {
			value, err := emitter.EmitExpression(spread.Value)
			if err != nil {
				return "", err
			}
			emitter.emitRuntimeArraySpread(array, value)
			continue
		}
		value, err := emitter.EmitExpression(argument)
		if err != nil {
			return "", err
		}
		emitter.emitRuntimeArrayPush(array, value)
	}
	return array, nil
}

func (emitter *ExpressionEmitter) emitRuntimeCallable(symbol string, callee string, thisValue string, arguments string) (string, error) {
	result := emitter.nextValueName()
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: callee},
		{IRType: runtimeValueIRType, Value: thisValue},
		{IRType: runtimeValueIRType, Value: arguments},
	}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(symbol, runtimeValueIRType, args)})
	emitter.body = append(emitter.body, RuntimeCall(result, runtimeValueIRType, symbol, args))
	return result, nil
}
