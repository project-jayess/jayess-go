package runtime

type ValueKind string

const (
	UndefinedValue ValueKind = "undefined"
	NullValue      ValueKind = "null"
	BooleanValue   ValueKind = "boolean"
	NumberValue    ValueKind = "number"
	StringValue    ValueKind = "string"
	ObjectValue    ValueKind = "object"
	ArrayValue     ValueKind = "array"
	FunctionValue  ValueKind = "function"
	NativeValue    ValueKind = "native"
)

func ValueKinds() []ValueKind {
	return []ValueKind{
		UndefinedValue,
		NullValue,
		BooleanValue,
		NumberValue,
		StringValue,
		ObjectValue,
		ArrayValue,
		FunctionValue,
		NativeValue,
	}
}

func IsValueKind(kind ValueKind) bool {
	for _, candidate := range ValueKinds() {
		if candidate == kind {
			return true
		}
	}
	return false
}
