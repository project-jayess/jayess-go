package runtime

type CompilerVector struct {
	values []Value
}

func NewCompilerVector(values ...Value) *CompilerVector {
	vector := &CompilerVector{}
	vector.values = append(vector.values, values...)
	return vector
}

func (vector *CompilerVector) Len() int {
	if vector == nil {
		return 0
	}
	return len(vector.values)
}

func (vector *CompilerVector) Push(value Value) int {
	vector.values = append(vector.values, value)
	return len(vector.values)
}

func (vector *CompilerVector) Get(index int) (Value, bool) {
	if vector == nil || index < 0 || index >= len(vector.values) {
		return Undefined(), false
	}
	return vector.values[index], true
}

func (vector *CompilerVector) Set(index int, value Value) bool {
	if vector == nil || index < 0 || index >= len(vector.values) {
		return false
	}
	vector.values[index] = value
	return true
}

func (vector *CompilerVector) Values() []Value {
	if vector == nil {
		return nil
	}
	return append([]Value(nil), vector.values...)
}
