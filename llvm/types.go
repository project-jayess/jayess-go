package llvm

import "strconv"

type Type struct {
	ir string
}

func I32() Type {
	return Type{ir: "i32"}
}

func Void() Type {
	return Type{ir: "void"}
}

func (typ Type) String() string {
	return typ.ir
}

type Constant struct {
	typ   Type
	value string
}

func ConstI32(value int) Constant {
	return Constant{typ: I32(), value: intString(value)}
}

func (constant Constant) Type() Type {
	return constant.typ
}

func (constant Constant) String() string {
	return constant.value
}

func intString(value int) string {
	return strconv.Itoa(value)
}
