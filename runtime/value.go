package runtime

type Value struct {
	kind    ValueKind
	boolean bool
	number  float64
	bigint  string
	text    string
	object  *Object
	array   *Array
	fn      *Function
	native  any
}

func Undefined() Value {
	return Value{kind: UndefinedValue}
}

func Null() Value {
	return Value{kind: NullValue}
}

func NewBoolean(value bool) Value {
	return Value{kind: BooleanValue, boolean: value}
}

func NewNumber(value float64) Value {
	return Value{kind: NumberValue, number: value}
}

func NewBigInt(value string) Value {
	return Value{kind: BigIntValue, bigint: value}
}

func NewString(value string) Value {
	return Value{kind: StringValue, text: value}
}

func NewObjectValue(object *Object) Value {
	if object == nil {
		object = NewObject()
	}
	return Value{kind: ObjectValue, object: object}
}

func NewArrayValue(array *Array) Value {
	if array == nil {
		array = NewArray()
	}
	return Value{kind: ArrayValue, array: array}
}

func NewFunctionValue(function *Function) Value {
	if function == nil {
		function = NewFunction("", nil)
	}
	return Value{kind: FunctionValue, fn: function}
}

func NewNativeValue(value any) Value {
	return Value{kind: NativeValue, native: value}
}

func (value Value) Kind() ValueKind {
	return value.kind
}

func (value Value) Bool() bool {
	return value.boolean
}

func (value Value) Number() float64 {
	return value.number
}

func (value Value) BigInt() string {
	return value.bigint
}

func (value Value) Text() string {
	return value.text
}

func (value Value) Object() (*Object, bool) {
	if value.kind != ObjectValue || value.object == nil {
		return nil, false
	}
	return value.object, true
}

func (value Value) Array() (*Array, bool) {
	if value.kind != ArrayValue || value.array == nil {
		return nil, false
	}
	return value.array, true
}

func (value Value) Function() (*Function, bool) {
	if value.kind != FunctionValue || value.fn == nil {
		return nil, false
	}
	return value.fn, true
}

func (value Value) Native() any {
	return value.native
}
