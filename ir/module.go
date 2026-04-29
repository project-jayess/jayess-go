package ir

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

type ValueKind string

const (
	ValueNumber    ValueKind = "number"
	ValueBigInt    ValueKind = "bigint"
	ValueBoolean   ValueKind = "boolean"
	ValueString    ValueKind = "string"
	ValueNull      ValueKind = "null"
	ValueUndefined ValueKind = "undefined"
	ValueArgsArray ValueKind = "args_array"
	ValueArray     ValueKind = "array"
	ValueObject    ValueKind = "object"
	ValueFunction  ValueKind = "function"
	ValueDynamic   ValueKind = "dynamic"
)

type BinaryOperator string
type ComparisonOperator string
type LogicalOperator string
type UnaryOperator string

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

type Module struct {
	SourcePath       string
	Globals          []VariableDecl
	ExternFunctions  []ExternFunction
	Classes          []ClassDecl
	Functions        []Function
	LifetimeEligible []LocalLifetimeClassification
	EligibleParams   []ParameterLifetimeClassification
}

type LocalLifetimeClassification struct {
	Function string
	Name     string
	Line     int
	Column   int
	Kind     DeclarationKind
	InLoop   bool
}

type ParameterLifetimeClassification struct {
	Function string
	Name     string
}

type ClassDecl struct {
	Name       string
	SuperClass string
	Fields     []ClassField
	Methods    []ClassMethod
}

type ClassField struct {
	Name           string
	Private        bool
	Static         bool
	HasInitializer bool
}

type ClassMethod struct {
	Name          string
	Private       bool
	Static        bool
	IsConstructor bool
	IsGetter      bool
	IsSetter      bool
	ParamCount    int
}

type Parameter struct {
	Name            string
	Kind            ValueKind
	Rest            bool
	Default         Expression
	CleanupEligible bool
}

type Function struct {
	Visibility    Visibility
	Name          string
	Line          int
	Column        int
	IsConstructor bool
	ReturnFresh   bool
	Params        []Parameter
	Body          []Statement
}

type ExternFunction struct {
	Name        string
	SymbolName  string
	BorrowsArgs bool
	Params      []Parameter
	Variadic    bool
}
