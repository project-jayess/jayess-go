package lowering

type returnScope struct {
	ints      map[string]int
	bigints   map[string]string
	bools     map[string]bool
	strings   map[string]string
	nullish   map[string]returnNullishKind
	funcs     map[string]int
	funcSeq   map[string]int
	objects   map[string]int
	objectSeq map[string]int
}

type returnNullishKind string

const (
	returnNullKind      returnNullishKind = "null"
	returnUndefinedKind returnNullishKind = "undefined"
)

func (scope returnScope) clone() returnScope {
	clone := returnScope{
		ints:      make(map[string]int, len(scope.ints)+1),
		bigints:   make(map[string]string, len(scope.bigints)+1),
		bools:     make(map[string]bool, len(scope.bools)+1),
		strings:   make(map[string]string, len(scope.strings)+1),
		nullish:   make(map[string]returnNullishKind, len(scope.nullish)+1),
		funcs:     make(map[string]int, len(scope.funcs)+1),
		funcSeq:   make(map[string]int, len(scope.funcSeq)+1),
		objects:   make(map[string]int, len(scope.objects)+1),
		objectSeq: make(map[string]int, len(scope.objectSeq)+1),
	}
	for name, value := range scope.ints {
		clone.ints[name] = value
	}
	for name, value := range scope.bigints {
		clone.bigints[name] = value
	}
	for name, value := range scope.bools {
		clone.bools[name] = value
	}
	for name, value := range scope.strings {
		clone.strings[name] = value
	}
	for name, value := range scope.nullish {
		clone.nullish[name] = value
	}
	for name, value := range scope.funcs {
		clone.funcs[name] = value
	}
	for name, value := range scope.funcSeq {
		clone.funcSeq[name] = value
	}
	for name, value := range scope.objects {
		clone.objects[name] = value
	}
	for name, value := range scope.objectSeq {
		clone.objectSeq[name] = value
	}
	return clone
}

func clearReturnScopeBinding(scope returnScope, name string) {
	delete(scope.ints, name)
	delete(scope.bigints, name)
	delete(scope.bools, name)
	delete(scope.strings, name)
	delete(scope.nullish, name)
	delete(scope.funcs, name)
	delete(scope.objects, name)
}

func replaceReturnScopeBindings(scope returnScope, next returnScope) {
	clearReturnScopeMap(scope.ints)
	clearReturnScopeMap(scope.bigints)
	clearReturnScopeMap(scope.bools)
	clearReturnScopeMap(scope.strings)
	clearReturnScopeMap(scope.nullish)
	clearReturnScopeMap(scope.funcs)
	clearReturnScopeMap(scope.funcSeq)
	clearReturnScopeMap(scope.objects)
	clearReturnScopeMap(scope.objectSeq)
	copyReturnScopeMap(scope.ints, next.ints)
	copyReturnScopeMap(scope.bigints, next.bigints)
	copyReturnScopeMap(scope.bools, next.bools)
	copyReturnScopeMap(scope.strings, next.strings)
	copyReturnScopeMap(scope.nullish, next.nullish)
	copyReturnScopeMap(scope.funcs, next.funcs)
	copyReturnScopeMap(scope.funcSeq, next.funcSeq)
	copyReturnScopeMap(scope.objects, next.objects)
	copyReturnScopeMap(scope.objectSeq, next.objectSeq)
}

func clearReturnScopeMap[K comparable, V any](values map[K]V) {
	for key := range values {
		delete(values, key)
	}
}

func copyReturnScopeMap[K comparable, V any](target map[K]V, source map[K]V) {
	for key, value := range source {
		target[key] = value
	}
}
