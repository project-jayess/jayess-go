package runtime

func ForInKeys(source Value) []string {
	switch source.Kind() {
	case ObjectValue:
		object, ok := source.Object()
		if ok {
			return object.Keys()
		}
	case ArrayValue:
		array, ok := source.Array()
		if ok {
			return array.Keys()
		}
	}
	return nil
}

func ForOfValues(source Value) []Value {
	switch source.Kind() {
	case ArrayValue:
		array, ok := source.Array()
		if ok {
			return array.Values()
		}
	case ObjectValue:
		object, ok := source.Object()
		if ok {
			return objectValues(object)
		}
	}
	return nil
}

func ForInKeyValues(source Value) []Value {
	keys := ForInKeys(source)
	values := make([]Value, 0, len(keys))
	for _, key := range keys {
		values = append(values, NewString(key))
	}
	return values
}

func objectValues(object *Object) []Value {
	keys := object.Keys()
	values := make([]Value, 0, len(keys))
	for _, key := range keys {
		value, _ := object.GetNamedProperty(key)
		values = append(values, value)
	}
	return values
}
