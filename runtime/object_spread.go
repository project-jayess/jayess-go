package runtime

func NewObjectFromSpread(sources ...Value) *Object {
	target := NewObject()
	for _, source := range sources {
		SpreadObjectInto(target, source)
	}
	return target
}

func SpreadObjectInto(target *Object, source Value) *Object {
	if target == nil {
		target = NewObject()
	}
	switch source.Kind() {
	case ObjectValue:
		object, ok := source.Object()
		if ok {
			copyObjectProperties(target, object, nil)
		}
	case ArrayValue:
		array, ok := source.Array()
		if ok {
			copyArrayProperties(target, array, nil)
		}
	}
	return target
}

func ObjectRest(source Value, excluded ...string) *Object {
	target := NewObject()
	excludedSet := stringSet(excluded)
	switch source.Kind() {
	case ObjectValue:
		object, ok := source.Object()
		if ok {
			copyObjectProperties(target, object, excludedSet)
		}
	case ArrayValue:
		array, ok := source.Array()
		if ok {
			copyArrayProperties(target, array, excludedSet)
		}
	}
	return target
}

func copyObjectProperties(target *Object, source *Object, excluded map[string]bool) {
	for _, key := range source.Keys() {
		if excluded[key] {
			continue
		}
		value, _ := source.GetNamedProperty(key)
		target.SetNamedProperty(key, value)
	}
}

func copyArrayProperties(target *Object, source *Array, excluded map[string]bool) {
	for _, key := range source.Keys() {
		if excluded[key] {
			continue
		}
		value, _ := source.GetNamedProperty(key)
		target.SetNamedProperty(key, value)
	}
}

func stringSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		set[value] = true
	}
	return set
}
