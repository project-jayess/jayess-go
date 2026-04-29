package ast

type Node interface {
	node()
}

type SourcePos struct {
	Line   int
	Column int
}

type BaseNode struct {
	Pos SourcePos
}

func (b BaseNode) SourcePosition() SourcePos {
	return b.Pos
}

type Positioned interface {
	SourcePosition() SourcePos
}

func PositionOf(value any) SourcePos {
	if positioned, ok := value.(Positioned); ok {
		return positioned.SourcePosition()
	}
	return SourcePos{}
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Pattern interface {
	Node
	patternNode()
}

type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

type DeclarationKind string

const (
	DeclarationVar   DeclarationKind = "var"
	DeclarationLet   DeclarationKind = "let"
	DeclarationConst DeclarationKind = "const"
)

type BinaryOperator string
type ComparisonOperator string
type LogicalOperator string
type UnaryOperator string
type AssignmentOperator string

const (
	OperatorAdd    BinaryOperator = "+"
	OperatorSub    BinaryOperator = "-"
	OperatorMul    BinaryOperator = "*"
	OperatorDiv    BinaryOperator = "/"
	OperatorBitAnd BinaryOperator = "&"
	OperatorBitOr  BinaryOperator = "|"
	OperatorBitXor BinaryOperator = "^"
	OperatorShl    BinaryOperator = "<<"
	OperatorShr    BinaryOperator = ">>"
	OperatorUShr   BinaryOperator = ">>>"
)

const (
	OperatorEq       ComparisonOperator = "=="
	OperatorNe       ComparisonOperator = "!="
	OperatorStrictEq ComparisonOperator = "==="
	OperatorStrictNe ComparisonOperator = "!=="
	OperatorLt       ComparisonOperator = "<"
	OperatorLte      ComparisonOperator = "<="
	OperatorGt       ComparisonOperator = ">"
	OperatorGte      ComparisonOperator = ">="
)

const (
	OperatorAnd LogicalOperator = "&&"
	OperatorOr  LogicalOperator = "||"
)

const (
	OperatorNot    UnaryOperator = "!"
	OperatorBitNot UnaryOperator = "~"
)

const (
	AssignmentAssign        AssignmentOperator = "="
	AssignmentAddAssign     AssignmentOperator = "+="
	AssignmentSubAssign     AssignmentOperator = "-="
	AssignmentMulAssign     AssignmentOperator = "*="
	AssignmentDivAssign     AssignmentOperator = "/="
	AssignmentNullishAssign AssignmentOperator = "??="
	AssignmentOrAssign      AssignmentOperator = "||="
	AssignmentAndAssign     AssignmentOperator = "&&="
)
