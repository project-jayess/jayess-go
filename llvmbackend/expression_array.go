package llvmbackend

import "jayess-go/ast"

const (
	runtimeArrayNewSymbol    = "jayess_array_new"
	runtimeArrayPushSymbol   = "jayess_array_push"
	runtimeArrayElideSymbol  = "jayess_array_elide"
	runtimeArraySpreadSymbol = "jayess_array_spread"
)

func (emitter *ExpressionEmitter) emitArrayLiteral(expression *ast.ArrayLiteral) (string, error) {
	array, err := emitter.emitRuntimeArrayNew()
	if err != nil {
		return "", err
	}
	for _, element := range expression.Elements {
		if element == nil {
			emitter.emitRuntimeArrayElision(array)
			continue
		}
		if spread, ok := element.(*ast.SpreadExpression); ok {
			value, err := emitter.EmitExpression(spread.Value)
			if err != nil {
				return "", err
			}
			emitter.emitRuntimeArraySpread(array, value)
			continue
		}
		value, err := emitter.EmitExpression(element)
		if err != nil {
			return "", err
		}
		emitter.emitRuntimeArrayPush(array, value)
	}
	return array, nil
}

func (emitter *ExpressionEmitter) emitRuntimeArrayNew() (string, error) {
	result := emitter.nextValueName()
	args := []RuntimeCallArg{}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeArrayNewSymbol, runtimeValueIRType, args)})
	emitter.body = append(emitter.body, RuntimeCall(result, runtimeValueIRType, runtimeArrayNewSymbol, args))
	return result, nil
}

func (emitter *ExpressionEmitter) emitRuntimeArrayPush(array string, value string) {
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: array},
		{IRType: runtimeValueIRType, Value: value},
	}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeArrayPushSymbol, "void", args)})
	emitter.body = append(emitter.body, RuntimeVoidCall(runtimeArrayPushSymbol, args))
}

func (emitter *ExpressionEmitter) emitRuntimeArrayElision(array string) {
	args := []RuntimeCallArg{{IRType: runtimeValueIRType, Value: array}}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeArrayElideSymbol, "void", args)})
	emitter.body = append(emitter.body, RuntimeVoidCall(runtimeArrayElideSymbol, args))
}

func (emitter *ExpressionEmitter) emitRuntimeArraySpread(array string, value string) {
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: array},
		{IRType: runtimeValueIRType, Value: value},
	}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeArraySpreadSymbol, "void", args)})
	emitter.body = append(emitter.body, RuntimeVoidCall(runtimeArraySpreadSymbol, args))
}
