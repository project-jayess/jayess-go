package runtime

func (class *Class) ResolveMethod(name string) (*Function, bool) {
	for current := class; current != nil; current = current.parent {
		if method, ok := current.methods[name]; ok {
			return method, true
		}
	}
	return nil, false
}

func (class *Class) ResolvePrivateMethod(name string) (*Function, bool) {
	for current := class; current != nil; current = current.parent {
		if method, ok := current.privateMethods[name]; ok {
			return method, true
		}
	}
	return nil, false
}

func (class *Class) ResolveSuperMethod(name string) (*Function, bool) {
	if class == nil || class.parent == nil {
		return nil, false
	}
	return class.parent.ResolveMethod(name)
}

func (object *Object) ResolveMethod(name string) (*Function, bool) {
	if object == nil || object.class == nil {
		return nil, false
	}
	return object.class.ResolveMethod(name)
}

func (object *Object) CallMethod(name string, arguments ...Value) Value {
	method, ok := object.ResolveMethod(name)
	if !ok {
		return Undefined()
	}
	return method.Call(NewObjectValue(object), arguments...)
}

func (object *Object) CallSuperMethod(name string, arguments ...Value) Value {
	if object == nil || object.class == nil {
		return Undefined()
	}
	method, ok := object.class.ResolveSuperMethod(name)
	if !ok {
		return Undefined()
	}
	return method.Call(NewObjectValue(object), arguments...)
}
