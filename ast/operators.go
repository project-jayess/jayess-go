package ast

type BinaryOperator string
type ComparisonOperator string
type LogicalOperator string
type UnaryOperator string
type UpdateOperator string
type AssignmentOperator string

const (
	OperatorAdd    BinaryOperator = "+"
	OperatorSub    BinaryOperator = "-"
	OperatorMul    BinaryOperator = "*"
	OperatorPow    BinaryOperator = "**"
	OperatorDiv    BinaryOperator = "/"
	OperatorMod    BinaryOperator = "%"
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
	OperatorIn       ComparisonOperator = "in"
)

const (
	OperatorAnd LogicalOperator = "&&"
	OperatorOr  LogicalOperator = "||"
)

const (
	OperatorNot      UnaryOperator = "!"
	OperatorBitNot   UnaryOperator = "~"
	OperatorNegate   UnaryOperator = "-"
	OperatorPositive UnaryOperator = "+"
	OperatorVoid     UnaryOperator = "void"
	OperatorDelete   UnaryOperator = "delete"
)

const (
	UpdateIncrement UpdateOperator = "++"
	UpdateDecrement UpdateOperator = "--"
)

const (
	AssignmentAssign        AssignmentOperator = "="
	AssignmentAddAssign     AssignmentOperator = "+="
	AssignmentSubAssign     AssignmentOperator = "-="
	AssignmentMulAssign     AssignmentOperator = "*="
	AssignmentPowAssign     AssignmentOperator = "**="
	AssignmentDivAssign     AssignmentOperator = "/="
	AssignmentModAssign     AssignmentOperator = "%="
	AssignmentBitAndAssign  AssignmentOperator = "&="
	AssignmentBitOrAssign   AssignmentOperator = "|="
	AssignmentBitXorAssign  AssignmentOperator = "^="
	AssignmentShlAssign     AssignmentOperator = "<<="
	AssignmentShrAssign     AssignmentOperator = ">>="
	AssignmentUShrAssign    AssignmentOperator = ">>>="
	AssignmentNullishAssign AssignmentOperator = "??="
	AssignmentOrAssign      AssignmentOperator = "||="
	AssignmentAndAssign     AssignmentOperator = "&&="
)
