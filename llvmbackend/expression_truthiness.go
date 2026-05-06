package llvmbackend

const runtimeTruthySymbol = "jayess_value_truthy"

func (emitter *ExpressionEmitter) EmitTruthiness(value string) (string, error) {
	result := emitter.nextValueName()
	args := []RuntimeCallArg{{IRType: runtimeValueIRType, Value: value}}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeTruthySymbol, "i1", args)})
	emitter.body = append(emitter.body, RuntimeCall(result, "i1", runtimeTruthySymbol, args))
	return result, nil
}
