package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
	jayessruntime "jayess-go/runtime"
)

const (
	runtimeObjectNewSymbol    = "jayess_object_new"
	runtimeObjectSpreadSymbol = "jayess_object_spread"
)

func (emitter *ExpressionEmitter) emitObjectLiteral(expression *ast.ObjectLiteral) (string, error) {
	object, err := emitter.emitRuntimeObjectNew()
	if err != nil {
		return "", err
	}
	for _, property := range expression.Properties {
		if property.Spread {
			value, err := emitter.EmitExpression(property.Value)
			if err != nil {
				return "", err
			}
			emitter.emitRuntimeObjectSpread(object, value)
			continue
		}
		key, err := emitter.emitObjectPropertyKey(property)
		if err != nil {
			return "", err
		}
		value, err := emitter.EmitExpression(property.Value)
		if err != nil {
			return "", err
		}
		if err := emitter.emitRuntimePropertyStore(object, key, value); err != nil {
			return "", err
		}
	}
	return object, nil
}

func (emitter *ExpressionEmitter) emitRuntimeObjectNew() (string, error) {
	result := emitter.nextValueName()
	args := []RuntimeCallArg{}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeObjectNewSymbol, runtimeValueIRType, args)})
	emitter.body = append(emitter.body, RuntimeCall(result, runtimeValueIRType, runtimeObjectNewSymbol, args))
	return result, nil
}

func (emitter *ExpressionEmitter) emitObjectPropertyKey(property ast.ObjectProperty) (string, error) {
	if property.Computed {
		if property.KeyExpr == nil {
			return "", fmt.Errorf("computed object property key must have an expression")
		}
		return emitter.EmitExpression(property.KeyExpr)
	}
	key := property.Key
	if key == "" && property.Shorthand {
		key = property.Key
	}
	return emitter.EmitRuntimeLiteral(RuntimeLiteral{Kind: jayessruntime.StringValue, Text: key})
}

func (emitter *ExpressionEmitter) emitRuntimePropertyStore(object string, key string, value string) error {
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: object},
		{IRType: runtimeValueIRType, Value: key},
		{IRType: runtimeValueIRType, Value: value},
	}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeSetPropertySymbol, "void", args)})
	emitter.body = append(emitter.body, RuntimeVoidCall(runtimeSetPropertySymbol, args))
	return nil
}

func (emitter *ExpressionEmitter) emitRuntimeObjectSpread(object string, value string) {
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: object},
		{IRType: runtimeValueIRType, Value: value},
	}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeObjectSpreadSymbol, "void", args)})
	emitter.body = append(emitter.body, RuntimeVoidCall(runtimeObjectSpreadSymbol, args))
}
