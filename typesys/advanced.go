package typesys

type AdvancedKind string

const (
	AdvancedUnion              AdvancedKind = "union"
	AdvancedIntersection       AdvancedKind = "intersection"
	AdvancedLiteral            AdvancedKind = "literal"
	AdvancedDiscriminatedUnion AdvancedKind = "discriminated_union"
	AdvancedGeneric            AdvancedKind = "generic"
	AdvancedConstraint         AdvancedKind = "constraint"
	AdvancedEnum               AdvancedKind = "enum"
)

type LiteralValue struct {
	TypeName string
	Value    string
}

type GenericParameter struct {
	Name       string
	Constraint string
	Default    string
}

type EnumMember struct {
	Name  string
	Value string
}

type AdvancedType struct {
	Name          string
	Kind          AdvancedKind
	Types         []string
	Literal       LiteralValue
	Discriminator string
	Variants      []StructuredType
	Parameters    []GenericParameter
	Body          StructuredType
	EnumMembers   []EnumMember
}

func UnionType(types ...string) AdvancedType {
	return AdvancedType{Kind: AdvancedUnion, Types: types}
}

func IntersectionType(types ...string) AdvancedType {
	return AdvancedType{Kind: AdvancedIntersection, Types: types}
}

func LiteralType(typeName string, value string) AdvancedType {
	return AdvancedType{Kind: AdvancedLiteral, Literal: LiteralValue{TypeName: typeName, Value: value}}
}

func DiscriminatedUnionType(discriminator string, variants []StructuredType) AdvancedType {
	return AdvancedType{Kind: AdvancedDiscriminatedUnion, Discriminator: discriminator, Variants: variants}
}

func GenericType(name string, parameters []GenericParameter, body StructuredType) AdvancedType {
	return AdvancedType{Name: name, Kind: AdvancedGeneric, Parameters: parameters, Body: body}
}

func GenericConstraint(name string, constraint string) AdvancedType {
	return AdvancedType{Name: name, Kind: AdvancedConstraint, Parameters: []GenericParameter{{Name: name, Constraint: constraint}}}
}

func EnumType(name string, members []EnumMember) AdvancedType {
	return AdvancedType{Name: name, Kind: AdvancedEnum, EnumMembers: members}
}
