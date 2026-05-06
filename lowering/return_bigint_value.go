package lowering

import (
	"math/big"
	"strings"

	"jayess-go/ast"
)

func evaluateBigIntTruthiness(expression ast.Expression) (bool, bool) {
	literal, ok := expression.(*ast.BigIntLiteral)
	if !ok {
		return false, false
	}
	return isTruthyBigIntValue(literal.Value), true
}

func evaluateBigIntStringCoercion(expression ast.Expression) (string, bool) {
	literal, ok := expression.(*ast.BigIntLiteral)
	if !ok {
		return "", false
	}
	return literal.Value, true
}

func evaluateBigIntLiteralValue(expression ast.Expression) (string, bool) {
	literal, ok := expression.(*ast.BigIntLiteral)
	if !ok {
		return "", false
	}
	return normalizeBigIntValue(literal.Value), true
}

func parseBigIntValue(value string) (*big.Int, bool) {
	parsed, ok := new(big.Int).SetString(value, 10)
	return parsed, ok
}

func normalizeBigIntValue(value string) string {
	sign := ""
	if strings.HasPrefix(value, "-") {
		sign = "-"
		value = strings.TrimPrefix(value, "-")
	}
	normalized := strings.TrimLeft(value, "0")
	if normalized == "" {
		return "0"
	}
	return sign + normalized
}

func isTruthyBigIntValue(value string) bool {
	return normalizeBigIntValue(value) != "0"
}
