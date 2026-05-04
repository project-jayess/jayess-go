package runtime

func (class *Class) Construct(arguments ...Value) *Object {
	instance := class.NewInstance()
	if class.constructor != nil {
		class.constructor.Call(NewObjectValue(instance), arguments...)
	}
	return instance
}

func (class *Class) ConstructSuper(instance *Object, arguments ...Value) Value {
	if class == nil || class.parent == nil || class.parent.constructor == nil {
		return Undefined()
	}
	if instance == nil {
		instance = class.NewInstance()
	}
	return class.parent.constructor.Call(NewObjectValue(instance), arguments...)
}
