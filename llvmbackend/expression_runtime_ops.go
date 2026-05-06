package llvmbackend

func (emitter *ExpressionEmitter) emitRuntimeUnaryValue(symbol string, value string) (string, error) {
	args := []RuntimeCallArg{{IRType: runtimeValueIRType, Value: value}}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(symbol, runtimeValueIRType, args)})
	result := emitter.nextValueName()
	emitter.body = append(emitter.body, RuntimeCall(result, runtimeValueIRType, symbol, args))
	return result, nil
}

func (emitter *ExpressionEmitter) emitRuntimeBinaryValue(symbol string, left string, right string) (string, error) {
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: left},
		{IRType: runtimeValueIRType, Value: right},
	}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(symbol, runtimeValueIRType, args)})
	result := emitter.nextValueName()
	emitter.body = append(emitter.body, RuntimeCall(result, runtimeValueIRType, symbol, args))
	return result, nil
}
