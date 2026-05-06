package llvmbackend

func (emitter *ExpressionEmitter) EmitRuntimeLiteral(literal RuntimeLiteral) (string, error) {
	result := emitter.nextValueName()
	lowered, err := LowerRuntimeLiteral(result, literal, emitter.stringIndex)
	if err != nil {
		return "", err
	}
	emitter.addDeclarations(lowered.Declarations)
	emitter.globals = append(emitter.globals, lowered.Globals...)
	emitter.stringIndex += len(lowered.Globals)
	emitter.body = append(emitter.body, lowered.Body...)
	return result, nil
}
