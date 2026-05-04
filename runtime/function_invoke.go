package runtime

func BindFunction(value Value, this Value, arguments ...Value) Value {
	function, ok := value.Function()
	if !ok {
		return Undefined()
	}
	return NewFunctionValue(NewBoundFunction(function, this, arguments...))
}

func CallMethod(value Value, this Value, arguments ...Value) Value {
	return CallFunction(value, this, arguments...)
}

func ApplyFunction(value Value, this Value, arguments Value) Value {
	return CallFunction(value, this, applyArguments(arguments)...)
}

func applyArguments(arguments Value) []Value {
	if array, ok := arguments.Array(); ok {
		return array.Values()
	}
	if arguments.Kind() == UndefinedValue || arguments.Kind() == NullValue {
		return nil
	}
	return []Value{arguments}
}
