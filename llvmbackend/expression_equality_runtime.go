package llvmbackend

const runtimeStrictEqualSymbol = "jayess_value_strict_equal"

func (emitter *ExpressionEmitter) EmitRuntimeStrictEqual(left string, right string) (string, error) {
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: left},
		{IRType: runtimeValueIRType, Value: right},
	}
	emitter.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeStrictEqualSymbol, "i1", args)})
	result := emitter.nextValueName()
	emitter.body = append(emitter.body, RuntimeCall(result, "i1", runtimeStrictEqualSymbol, args))
	return result, nil
}
