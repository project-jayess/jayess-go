package runtime

type CallFrame struct {
	this        Value
	arguments   []Value
	environment *ClosureEnvironment
}

func NewCallFrame(this Value, arguments ...Value) CallFrame {
	return NewCallFrameWithClosure(this, nil, arguments...)
}

func NewCallFrameWithClosure(this Value, environment *ClosureEnvironment, arguments ...Value) CallFrame {
	copied := make([]Value, len(arguments))
	copy(copied, arguments)
	return CallFrame{this: this, arguments: copied, environment: environment}
}

func (frame CallFrame) This() Value {
	return frame.this
}

func (frame CallFrame) Closure() (*ClosureEnvironment, bool) {
	if frame.environment == nil {
		return nil, false
	}
	return frame.environment, true
}

func (frame CallFrame) Argument(index int) Value {
	if index < 0 || index >= len(frame.arguments) {
		return Undefined()
	}
	return frame.arguments[index]
}

func (frame CallFrame) HasArgument(index int) bool {
	return index >= 0 && index < len(frame.arguments) && frame.arguments[index].Kind() != UndefinedValue
}

func (frame CallFrame) ArgumentOrDefault(index int, fallback Value) Value {
	if !frame.HasArgument(index) {
		return fallback
	}
	return frame.arguments[index]
}

func (frame CallFrame) ArgumentCount() int {
	return len(frame.arguments)
}

func (frame CallFrame) Arguments() []Value {
	copied := make([]Value, len(frame.arguments))
	copy(copied, frame.arguments)
	return copied
}

func (frame CallFrame) RestArguments(start int) *Array {
	if start < 0 {
		start = 0
	}
	rest := NewArray()
	for index := start; index < len(frame.arguments); index++ {
		rest.Push(frame.arguments[index])
	}
	return rest
}
