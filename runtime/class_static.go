package runtime

func (class *Class) DefineStaticField(name string, value Value) {
	if class.staticFields == nil {
		class.staticFields = map[string]Value{}
	}
	if _, exists := class.staticFields[name]; !exists {
		class.staticOrder = append(class.staticOrder, name)
	}
	class.staticFields[name] = value
}

func (class *Class) StaticField(name string) (Value, bool) {
	if class.staticFields == nil {
		return Undefined(), false
	}
	value, exists := class.staticFields[name]
	if !exists {
		return Undefined(), false
	}
	return value, true
}

func (class *Class) StaticFieldNames() []string {
	names := make([]string, 0, len(class.staticOrder))
	for _, name := range class.staticOrder {
		if _, exists := class.staticFields[name]; exists {
			names = append(names, name)
		}
	}
	return names
}

func (class *Class) DefineStaticBlock(block *Function) {
	class.staticBlocks = append(class.staticBlocks, block)
}

func (class *Class) RunStaticBlocks() {
	if class.staticDone {
		return
	}
	class.staticDone = true
	this := NewNativeValue(class)
	for _, block := range class.staticBlocks {
		if block != nil {
			block.Call(this)
		}
	}
}

func (class *Class) StaticBlocksRan() bool {
	return class.staticDone
}
