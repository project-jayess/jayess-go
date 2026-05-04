package runtime

type ValueRuntimeSymbol struct {
	Name   string
	Kind   ValueKind
	Result ValueKind
}

func ValueRuntimeSymbols() []ValueRuntimeSymbol {
	return []ValueRuntimeSymbol{
		{Name: "jayess_value_undefined", Kind: UndefinedValue, Result: UndefinedValue},
		{Name: "jayess_value_null", Kind: NullValue, Result: NullValue},
		{Name: "jayess_value_from_boolean", Kind: BooleanValue, Result: BooleanValue},
		{Name: "jayess_value_from_number", Kind: NumberValue, Result: NumberValue},
		{Name: "jayess_value_from_string_copy", Kind: StringValue, Result: StringValue},
		{Name: "jayess_object_new", Kind: ObjectValue, Result: ObjectValue},
		{Name: "jayess_array_new", Kind: ArrayValue, Result: ArrayValue},
		{Name: "jayess_function_new", Kind: FunctionValue, Result: FunctionValue},
		{Name: "jayess_value_from_native_handle", Kind: NativeValue, Result: NativeValue},
	}
}

func HasValueRuntimeSymbol(name string) bool {
	for _, symbol := range ValueRuntimeSymbols() {
		if symbol.Name == name {
			return true
		}
	}
	return false
}
