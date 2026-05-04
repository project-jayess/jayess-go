package runtime

type arraySlot struct {
	value   Value
	present bool
}

type Array struct {
	slots      []arraySlot
	properties *Object
}

func NewArray(values ...Value) *Array {
	array := &Array{properties: NewObject()}
	for _, value := range values {
		array.Push(value)
	}
	return array
}

func (array *Array) Length() int {
	return len(array.slots)
}

func (array *Array) Push(value Value) int {
	array.slots = append(array.slots, arraySlot{value: value, present: true})
	return array.Length()
}

func (array *Array) SetIndex(index int, value Value) {
	if index < 0 {
		return
	}
	array.ensureLength(index + 1)
	array.slots[index] = arraySlot{value: value, present: true}
}

func (array *Array) GetIndex(index int) (Value, bool) {
	if index < 0 || index >= len(array.slots) || !array.slots[index].present {
		return Undefined(), false
	}
	return array.slots[index].value, true
}

func (array *Array) DeleteIndex(index int) bool {
	if index < 0 || index >= len(array.slots) || !array.slots[index].present {
		return false
	}
	array.slots[index] = arraySlot{}
	return true
}

func (array *Array) SetLength(length int) {
	if length < 0 {
		length = 0
	}
	array.ensureLength(length)
	array.slots = array.slots[:length]
}

func (array *Array) Values() []Value {
	values := make([]Value, len(array.slots))
	for index, slot := range array.slots {
		if slot.present {
			values[index] = slot.value
			continue
		}
		values[index] = Undefined()
	}
	return values
}

func (array *Array) Keys() []string {
	keys := make([]string, 0, len(array.slots))
	for index, slot := range array.slots {
		if slot.present {
			keys = append(keys, arrayIndexKey(index))
		}
	}
	array.ensureProperties()
	keys = append(keys, array.properties.Keys()...)
	return keys
}

func (array *Array) SetProperty(key Value, value Value) {
	array.SetNamedProperty(PropertyKey(key), value)
}

func (array *Array) SetNamedProperty(key string, value Value) {
	if key == "length" {
		array.SetLength(int(value.Number()))
		return
	}
	if index, ok := arrayIndexFromKey(key); ok {
		array.SetIndex(index, value)
		return
	}
	array.ensureProperties()
	array.properties.SetNamedProperty(key, value)
}

func (array *Array) GetProperty(key Value) (Value, bool) {
	return array.GetNamedProperty(PropertyKey(key))
}

func (array *Array) GetNamedProperty(key string) (Value, bool) {
	if key == "length" {
		return NewNumber(float64(array.Length())), true
	}
	if index, ok := arrayIndexFromKey(key); ok {
		return array.GetIndex(index)
	}
	array.ensureProperties()
	return array.properties.GetNamedProperty(key)
}

func (array *Array) HasNamedProperty(key string) bool {
	if key == "length" {
		return true
	}
	if index, ok := arrayIndexFromKey(key); ok {
		_, exists := array.GetIndex(index)
		return exists
	}
	array.ensureProperties()
	return array.properties.HasNamedProperty(key)
}

func (array *Array) ensureLength(length int) {
	for len(array.slots) < length {
		array.slots = append(array.slots, arraySlot{})
	}
}

func (array *Array) ensureProperties() {
	if array.properties == nil {
		array.properties = NewObject()
	}
}
