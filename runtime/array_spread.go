package runtime

func NewArrayFromElements(values ...Value) *Array {
	array := NewArray()
	for _, value := range values {
		array.Push(value)
	}
	return array
}

func NewArrayFromSpread(sources ...Value) *Array {
	array := NewArray()
	for _, source := range sources {
		SpreadArrayInto(array, source)
	}
	return array
}

func SpreadArrayInto(target *Array, source Value) *Array {
	if target == nil {
		target = NewArray()
	}
	switch source.Kind() {
	case ArrayValue:
		array, ok := source.Array()
		if ok {
			for _, value := range array.Values() {
				target.Push(value)
			}
		}
	default:
		target.Push(source)
	}
	return target
}

func ArrayRest(source Value, start int) *Array {
	target := NewArray()
	array, ok := source.Array()
	if !ok {
		return target
	}
	if start < 0 {
		start = 0
	}
	values := array.Values()
	for index := start; index < len(values); index++ {
		target.Push(values[index])
	}
	return target
}
