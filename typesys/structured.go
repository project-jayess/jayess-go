package typesys

type StructuredKind string

const (
	StructuredInterface      StructuredKind = "interface"
	StructuredAlias          StructuredKind = "alias"
	StructuredObject         StructuredKind = "object"
	StructuredFunction       StructuredKind = "function"
	StructuredCallable       StructuredKind = "callable"
	StructuredIndexSignature StructuredKind = "index_signature"
)

type Property struct {
	Name     string
	TypeName string
	Optional bool
	Readonly bool
}

type Parameter struct {
	Name     string
	TypeName string
	Optional bool
}

type IndexSignature struct {
	KeyName   string
	KeyType   string
	ValueType string
	Readonly  bool
}

type StructuredType struct {
	Name            string
	Kind            StructuredKind
	Target          string
	Properties      []Property
	Parameters      []Parameter
	ReturnType      string
	IndexSignatures []IndexSignature
}

func InterfaceType(name string, properties []Property) StructuredType {
	return StructuredType{Name: name, Kind: StructuredInterface, Properties: properties}
}

func AliasType(name string, target string) StructuredType {
	return StructuredType{Name: name, Kind: StructuredAlias, Target: target}
}

func FunctionType(parameters []Parameter, returnType string) StructuredType {
	return StructuredType{Kind: StructuredFunction, Parameters: parameters, ReturnType: returnType}
}

func CallableType(parameters []Parameter, returnType string) StructuredType {
	return StructuredType{Kind: StructuredCallable, Parameters: parameters, ReturnType: returnType}
}

func ObjectType(properties []Property, indexSignatures []IndexSignature) StructuredType {
	return StructuredType{Kind: StructuredObject, Properties: properties, IndexSignatures: indexSignatures}
}
