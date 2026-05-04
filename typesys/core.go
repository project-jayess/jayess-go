package typesys

type Kind string

const (
	KindPrimitive Kind = "primitive"
	KindTop       Kind = "top"
	KindBottom    Kind = "bottom"
	KindVoid      Kind = "void"
	KindNullish   Kind = "nullish"
	KindObject    Kind = "object"
	KindArray     Kind = "array"
	KindTuple     Kind = "tuple"
)

type CoreType struct {
	Name string
	Kind Kind
}

func CoreTypes() []CoreType {
	return []CoreType{
		{Name: "number", Kind: KindPrimitive},
		{Name: "string", Kind: KindPrimitive},
		{Name: "boolean", Kind: KindPrimitive},
		{Name: "bigint", Kind: KindPrimitive},
		{Name: "void", Kind: KindVoid},
		{Name: "null", Kind: KindNullish},
		{Name: "undefined", Kind: KindNullish},
		{Name: "any", Kind: KindTop},
		{Name: "unknown", Kind: KindTop},
		{Name: "never", Kind: KindBottom},
		{Name: "object", Kind: KindObject},
		{Name: "array", Kind: KindArray},
		{Name: "tuple", Kind: KindTuple},
	}
}

func LookupCoreType(name string) (CoreType, bool) {
	for _, coreType := range CoreTypes() {
		if coreType.Name == name {
			return coreType, true
		}
	}
	return CoreType{}, false
}

func HasCoreType(name string) bool {
	_, ok := LookupCoreType(name)
	return ok
}
