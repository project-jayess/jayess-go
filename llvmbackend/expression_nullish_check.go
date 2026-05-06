package llvmbackend

const runtimeNullishSymbol = "jayess_value_is_nullish"

func (emitter *ExpressionEmitter) EmitNullishCheck(value string) (string, error) {
	result := emitter.nextValueName()
	args := []RuntimeCallArg{{IRType: runtimeValueIRType, Value: value}}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeNullishSymbol, "i1", args)})
	emitter.body = append(emitter.body, RuntimeCall(result, "i1", runtimeNullishSymbol, args))
	return result, nil
}
