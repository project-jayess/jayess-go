package runtime

type Accessor struct {
	Getter *Function
	Setter *Function
}

type Class struct {
	name           string
	parent         *Class
	constructor    *Function
	fields         map[string]Value
	privateFields  map[string]Value
	methods        map[string]*Function
	privateMethods map[string]*Function
	accessors      map[string]Accessor
	staticFields   map[string]Value
	staticOrder    []string
	staticBlocks   []*Function
	staticDone     bool
}

func NewClass(name string) *Class {
	return &Class{
		name:           name,
		fields:         map[string]Value{},
		privateFields:  map[string]Value{},
		methods:        map[string]*Function{},
		privateMethods: map[string]*Function{},
		accessors:      map[string]Accessor{},
		staticFields:   map[string]Value{},
	}
}

func NewClassExtends(name string, parent *Class) *Class {
	class := NewClass(name)
	class.parent = parent
	return class
}

func (class *Class) Name() string {
	return class.name
}

func (class *Class) Parent() (*Class, bool) {
	if class.parent == nil {
		return nil, false
	}
	return class.parent, true
}

func (class *Class) DefineConstructor(constructor *Function) {
	class.constructor = constructor
}

func (class *Class) Constructor() (*Function, bool) {
	if class.constructor == nil {
		return nil, false
	}
	return class.constructor, true
}

func (class *Class) DefineField(name string, value Value) {
	class.fields[name] = value
}

func (class *Class) DefinePrivateField(name string, value Value) {
	class.privateFields[name] = value
}

func (class *Class) DefineMethod(name string, method *Function) {
	class.methods[name] = method
}

func (class *Class) DefinePrivateMethod(name string, method *Function) {
	class.privateMethods[name] = method
}

func (class *Class) DefineAccessor(name string, getter *Function, setter *Function) {
	class.accessors[name] = Accessor{Getter: getter, Setter: setter}
}

func (class *Class) NewInstance() *Object {
	instance := NewObject()
	instance.class = class
	class.initializeInstance(instance)
	return instance
}

func (class *Class) initializeInstance(instance *Object) {
	if class.parent != nil {
		class.parent.initializeInstance(instance)
	}
	for name, value := range class.fields {
		instance.SetNamedProperty(name, value)
	}
	for name, value := range class.privateFields {
		instance.SetPrivateField(name, value)
	}
}

func (class *Class) Method(name string) (*Function, bool) {
	method, exists := class.methods[name]
	return method, exists
}

func (class *Class) PrivateMethod(name string) (*Function, bool) {
	method, exists := class.privateMethods[name]
	return method, exists
}

func (class *Class) Accessor(name string) (Accessor, bool) {
	accessor, exists := class.accessors[name]
	return accessor, exists
}
