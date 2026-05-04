package runtime

func (class *Class) ResolveAccessor(name string) (Accessor, bool) {
	for current := class; current != nil; current = current.parent {
		if accessor, ok := current.accessors[name]; ok {
			return accessor, true
		}
	}
	return Accessor{}, false
}

func (object *Object) GetAccessorProperty(name string) (Value, bool) {
	if object == nil || object.class == nil {
		return Undefined(), false
	}
	accessor, ok := object.class.ResolveAccessor(name)
	if !ok || accessor.Getter == nil {
		return Undefined(), false
	}
	return accessor.Getter.Call(NewObjectValue(object)), true
}

func (object *Object) SetAccessorProperty(name string, value Value) bool {
	if object == nil || object.class == nil {
		return false
	}
	accessor, ok := object.class.ResolveAccessor(name)
	if !ok || accessor.Setter == nil {
		return false
	}
	accessor.Setter.Call(NewObjectValue(object), value)
	return true
}

func (object *Object) GetCheckedPrivateField(owner *Class, name string) (Value, bool) {
	if !object.CanAccessPrivate(owner, name) {
		return Undefined(), false
	}
	return object.GetPrivateField(name)
}

func (object *Object) SetCheckedPrivateField(owner *Class, name string, value Value) bool {
	if !object.CanAccessPrivate(owner, name) {
		return false
	}
	object.SetPrivateField(name, value)
	return true
}

func (object *Object) CanAccessPrivate(owner *Class, name string) bool {
	if object == nil || owner == nil || object.class == nil {
		return false
	}
	if !object.class.extendsOrEquals(owner) {
		return false
	}
	return object.HasPrivateField(name)
}

func (class *Class) extendsOrEquals(owner *Class) bool {
	for current := class; current != nil; current = current.parent {
		if current == owner {
			return true
		}
	}
	return false
}
