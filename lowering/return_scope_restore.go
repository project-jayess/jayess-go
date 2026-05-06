package lowering

func restoreReturnScopeBinding(name string, source returnScope, target returnScope) {
	clearReturnScopeBinding(target, name)
	delete(target.funcSeq, name)
	delete(target.objectSeq, name)
	if value, ok := source.ints[name]; ok {
		target.ints[name] = value
	}
	if value, ok := source.bigints[name]; ok {
		target.bigints[name] = value
	}
	if value, ok := source.bools[name]; ok {
		target.bools[name] = value
	}
	if value, ok := source.strings[name]; ok {
		target.strings[name] = value
	}
	if value, ok := source.nullish[name]; ok {
		target.nullish[name] = value
	}
	if value, ok := source.funcs[name]; ok {
		target.funcs[name] = value
	}
	if value, ok := source.funcSeq[name]; ok {
		target.funcSeq[name] = value
	}
	if value, ok := source.objects[name]; ok {
		target.objects[name] = value
	}
	if value, ok := source.objectSeq[name]; ok {
		target.objectSeq[name] = value
	}
}
