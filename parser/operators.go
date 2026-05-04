package parser

import (
	"jayess-go/ast"
	"jayess-go/lexer"
)

func binaryOperator(tokenType lexer.TokenType) ast.BinaryOperator {
	switch tokenType {
	case lexer.TokenPlus:
		return ast.OperatorAdd
	case lexer.TokenMinus:
		return ast.OperatorSub
	case lexer.TokenStar:
		return ast.OperatorMul
	case lexer.TokenPower:
		return ast.OperatorPow
	case lexer.TokenSlash:
		return ast.OperatorDiv
	case lexer.TokenPercent:
		return ast.OperatorMod
	case lexer.TokenBitAnd:
		return ast.OperatorBitAnd
	case lexer.TokenBitOr:
		return ast.OperatorBitOr
	case lexer.TokenBitXor:
		return ast.OperatorBitXor
	case lexer.TokenShiftLeft:
		return ast.OperatorShl
	case lexer.TokenShiftRight:
		return ast.OperatorShr
	case lexer.TokenUnsignedShift:
		return ast.OperatorUShr
	default:
		return ""
	}
}

func comparisonOperator(tokenType lexer.TokenType) ast.ComparisonOperator {
	switch tokenType {
	case lexer.TokenEq:
		return ast.OperatorEq
	case lexer.TokenNe:
		return ast.OperatorNe
	case lexer.TokenStrictEq:
		return ast.OperatorStrictEq
	case lexer.TokenStrictNe:
		return ast.OperatorStrictNe
	case lexer.TokenLt:
		return ast.OperatorLt
	case lexer.TokenLte:
		return ast.OperatorLte
	case lexer.TokenGt:
		return ast.OperatorGt
	case lexer.TokenGte:
		return ast.OperatorGte
	case lexer.TokenIn:
		return ast.OperatorIn
	default:
		return ""
	}
}
