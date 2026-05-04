package binding

type HeaderFunction struct {
	Name     string
	Category string
}

func RuntimeHeaderFunctions() []HeaderFunction {
	return []HeaderFunction{
		{Name: "jayess_value_from_number", Category: "value"},
		{Name: "jayess_value_to_number", Category: "value"},
		{Name: "jayess_value_from_string_copy", Category: "string"},
		{Name: "jayess_value_to_string_copy", Category: "string"},
		{Name: "jayess_string_free", Category: "string"},
		{Name: "jayess_value_from_bytes_copy", Category: "bytes"},
		{Name: "jayess_value_to_bytes_copy", Category: "bytes"},
		{Name: "jayess_bytes_free", Category: "bytes"},
		{Name: "jayess_expect_object", Category: "object"},
		{Name: "jayess_expect_array", Category: "array"},
		{Name: "jayess_value_from_managed_native_handle", Category: "native-handle"},
		{Name: "jayess_value_close_native_handle", Category: "native-handle"},
		{Name: "jayess_throw_error", Category: "error"},
		{Name: "jayess_throw_type_error", Category: "error"},
	}
}

func RuntimeHeaderHasFunction(name string) bool {
	for _, fn := range RuntimeHeaderFunctions() {
		if fn.Name == name {
			return true
		}
	}
	return false
}
