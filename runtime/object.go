package runtime

type Object struct {
	properties map[string]Value
	order      []string
	class      *Class
	private    map[string]Value
}

func NewObject() *Object {
	return &Object{properties: map[string]Value{}}
}

func (object *Object) SetProperty(key Value, value Value) {
	object.SetNamedProperty(PropertyKey(key), value)
}

func (object *Object) SetNamedProperty(key string, value Value) {
	if object.properties == nil {
		object.properties = map[string]Value{}
	}
	if _, exists := object.properties[key]; !exists {
		object.order = append(object.order, key)
	}
	object.properties[key] = value
}

func (object *Object) GetProperty(key Value) (Value, bool) {
	return object.GetNamedProperty(PropertyKey(key))
}

func (object *Object) GetNamedProperty(key string) (Value, bool) {
	if object.properties == nil {
		return Undefined(), false
	}
	value, exists := object.properties[key]
	if !exists {
		return Undefined(), false
	}
	return value, true
}

func (object *Object) HasProperty(key Value) bool {
	return object.HasNamedProperty(PropertyKey(key))
}

func (object *Object) HasNamedProperty(key string) bool {
	if object.properties == nil {
		return false
	}
	_, exists := object.properties[key]
	return exists
}

func (object *Object) DeleteProperty(key Value) bool {
	return object.DeleteNamedProperty(PropertyKey(key))
}

func (object *Object) DeleteNamedProperty(key string) bool {
	if object.properties == nil {
		return false
	}
	if _, exists := object.properties[key]; !exists {
		return false
	}
	delete(object.properties, key)
	object.removeKey(key)
	return true
}

func (object *Object) Keys() []string {
	keys := make([]string, 0, len(object.order))
	for _, key := range object.order {
		if object.HasNamedProperty(key) {
			keys = append(keys, key)
		}
	}
	return keys
}

func (object *Object) Clone() *Object {
	clone := NewObject()
	clone.class = object.class
	for _, key := range object.Keys() {
		value, _ := object.GetNamedProperty(key)
		clone.SetNamedProperty(key, value)
	}
	for _, key := range object.PrivateKeys() {
		value, _ := object.GetPrivateField(key)
		clone.SetPrivateField(key, value)
	}
	return clone
}

func (object *Object) Class() (*Class, bool) {
	if object.class == nil {
		return nil, false
	}
	return object.class, true
}

func (object *Object) SetPrivateField(key string, value Value) {
	if object.private == nil {
		object.private = map[string]Value{}
	}
	object.private[key] = value
}

func (object *Object) GetPrivateField(key string) (Value, bool) {
	if object.private == nil {
		return Undefined(), false
	}
	value, exists := object.private[key]
	if !exists {
		return Undefined(), false
	}
	return value, true
}

func (object *Object) HasPrivateField(key string) bool {
	if object.private == nil {
		return false
	}
	_, exists := object.private[key]
	return exists
}

func (object *Object) PrivateKeys() []string {
	keys := make([]string, 0, len(object.private))
	for key := range object.private {
		keys = append(keys, key)
	}
	return keys
}

func (object *Object) removeKey(key string) {
	for index, candidate := range object.order {
		if candidate != key {
			continue
		}
		copy(object.order[index:], object.order[index+1:])
		object.order = object.order[:len(object.order)-1]
		return
	}
}
