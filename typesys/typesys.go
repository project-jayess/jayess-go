package typesys

type Kind int

const (
	KindAny Kind = iota
	KindSimple
	KindLiteral
	KindUnion
	KindIntersection
	KindTuple
	KindObject
	KindFunction
	KindApplication
)

type Expr struct {
	Kind            Kind
	Name            string
	Elements        []*Expr
	Properties      []Property
	IndexSignatures []IndexSignature
	Params          []*Expr
	Return          *Expr
	TypeArgs        []*Expr
}

type Property struct {
	Name     string
	Optional bool
	Readonly bool
	Type     *Expr
}

type IndexSignature struct {
	KeyName   string
	KeyType   *Expr
	ValueType *Expr
	Readonly  bool
}
