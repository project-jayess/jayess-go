package llvmbackend

import (
	"fmt"
	"strconv"

	"jayess-go/ast"
	jayessruntime "jayess-go/runtime"
)

func LowerASTRuntimeLiteral(result string, expression ast.Expression, stringIndex int) (RuntimeLiteralLowering, error) {
	literal, err := RuntimeLiteralFromAST(expression)
	if err != nil {
		return RuntimeLiteralLowering{}, err
	}
	return LowerRuntimeLiteral(result, literal, stringIndex)
}

func RuntimeLiteralFromAST(expression ast.Expression) (RuntimeLiteral, error) {
	switch expr := expression.(type) {
	case *ast.NumberLiteral:
		value, err := strconv.ParseFloat(expr.Value, 64)
		if err != nil {
			return RuntimeLiteral{}, fmt.Errorf("invalid number literal %q", expr.Value)
		}
		return RuntimeLiteral{Kind: jayessruntime.NumberValue, Number: value}, nil
	case *ast.BigIntLiteral:
		return RuntimeLiteral{Kind: jayessruntime.BigIntValue, Text: expr.Value}, nil
	case *ast.StringLiteral:
		return RuntimeLiteral{Kind: jayessruntime.StringValue, Text: expr.Value}, nil
	case *ast.BooleanLiteral:
		return RuntimeLiteral{Kind: jayessruntime.BooleanValue, Bool: expr.Value}, nil
	case *ast.NullLiteral:
		return RuntimeLiteral{Kind: jayessruntime.NullValue}, nil
	case *ast.UndefinedLiteral:
		return RuntimeLiteral{Kind: jayessruntime.UndefinedValue}, nil
	default:
		return RuntimeLiteral{}, fmt.Errorf("unsupported AST runtime literal %T", expression)
	}
}
