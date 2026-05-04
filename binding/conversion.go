package binding

type ValueKind string

const (
	NumberValue  ValueKind = "number"
	StringValue  ValueKind = "string"
	BooleanValue ValueKind = "boolean"
	NullishValue ValueKind = "nullish"
	ObjectValue  ValueKind = "object"
	ArrayValue   ValueKind = "array"
	BufferValue  ValueKind = "buffer"
	NativeHandle ValueKind = "native-handle"
)

type ConversionRule struct {
	Value       ValueKind
	Native      string
	ToNative    string
	FromNative  string
	Ownership   OwnershipRule
	TypeChecked bool
}

func ConversionRules() []ConversionRule {
	return []ConversionRule{
		{Value: NumberValue, Native: "double", ToNative: "jayess_value_to_number", FromNative: "jayess_value_from_number", TypeChecked: true},
		{Value: StringValue, Native: "char *", ToNative: "jayess_value_to_string_copy", FromNative: "jayess_value_from_string_copy", Ownership: CopiedStringsForStorage, TypeChecked: true},
		{Value: BooleanValue, Native: "bool", ToNative: "jayess_value_to_bool", FromNative: "jayess_value_from_bool", TypeChecked: true},
		{Value: NullishValue, Native: "NULL", ToNative: "jayess_value_is_nullish", FromNative: "jayess_value_from_null", TypeChecked: true},
		{Value: ObjectValue, Native: "jayess_object *", ToNative: "jayess_expect_object", FromNative: "jayess_value_from_object", Ownership: BorrowedViewsDuringCall, TypeChecked: true},
		{Value: ArrayValue, Native: "jayess_array *", ToNative: "jayess_expect_array", FromNative: "jayess_value_from_array", Ownership: BorrowedViewsDuringCall, TypeChecked: true},
		{Value: BufferValue, Native: "jayess_bytes", ToNative: "jayess_expect_bytes_copy", FromNative: "jayess_value_from_bytes_copy", Ownership: CopiedBytesForStorage, TypeChecked: true},
		{Value: NativeHandle, Native: "void *", ToNative: "jayess_expect_native_handle", FromNative: "jayess_value_from_managed_native_handle", Ownership: ManagedHandlesClosable, TypeChecked: true},
	}
}

func ConversionRuleFor(value ValueKind) (ConversionRule, bool) {
	for _, rule := range ConversionRules() {
		if rule.Value == value {
			return rule, true
		}
	}
	return ConversionRule{}, false
}
