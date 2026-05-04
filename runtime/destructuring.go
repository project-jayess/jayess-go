package runtime

func DestructureObjectProperty(source Value, key string, fallback Value) Value {
	switch source.Kind() {
	case ObjectValue:
		object, ok := source.Object()
		if ok {
			value, exists := object.GetNamedProperty(key)
			return valueOrFallback(value, exists, fallback)
		}
	case ArrayValue:
		array, ok := source.Array()
		if ok {
			value, exists := array.GetNamedProperty(key)
			return valueOrFallback(value, exists, fallback)
		}
	}
	return fallback
}

func DestructureArrayIndex(source Value, index int, fallback Value) Value {
	array, ok := source.Array()
	if !ok {
		return fallback
	}
	value, exists := array.GetIndex(index)
	return valueOrFallback(value, exists, fallback)
}

func valueOrFallback(value Value, ok bool, fallback Value) Value {
	if !ok || value.Kind() == UndefinedValue {
		return fallback
	}
	return value
}
