package test

import (
	"testing"

	"jayess-go/ast"
	"jayess-go/lexer"
	"jayess-go/parser"
)

func TestParserExpressionPrecedenceAndGrouping(t *testing.T) {
	expr := parseExpression(t, "(1 + 2) * 3 == 9 && true")

	logical := requireType[*ast.LogicalExpression](t, expr)
	if logical.Operator != ast.OperatorAnd {
		t.Fatalf("expected && at root, got %q", logical.Operator)
	}
	compare := requireType[*ast.ComparisonExpression](t, logical.Left)
	if compare.Operator != ast.OperatorEq {
		t.Fatalf("expected == comparison, got %q", compare.Operator)
	}
	multiply := requireType[*ast.BinaryExpression](t, compare.Left)
	if multiply.Operator != ast.OperatorMul {
		t.Fatalf("expected grouped multiplication, got %q", multiply.Operator)
	}
	add := requireType[*ast.BinaryExpression](t, multiply.Left)
	if add.Operator != ast.OperatorAdd {
		t.Fatalf("expected grouped addition on left side, got %q", add.Operator)
	}
}

func TestParserExpressionModuloPrecedence(t *testing.T) {
	expr := parseExpression(t, "value + count % 2 * size")
	add := requireType[*ast.BinaryExpression](t, expr)
	if add.Operator != ast.OperatorAdd {
		t.Fatalf("expected + at root, got %q", add.Operator)
	}
	multiply := requireType[*ast.BinaryExpression](t, add.Right)
	if multiply.Operator != ast.OperatorMul {
		t.Fatalf("expected * on right side, got %q", multiply.Operator)
	}
	modulo := requireType[*ast.BinaryExpression](t, multiply.Left)
	if modulo.Operator != ast.OperatorMod {
		t.Fatalf("expected modulo before multiplication, got %q", modulo.Operator)
	}
}

func TestParserExpressionExponentiationPrecedence(t *testing.T) {
	expr := parseExpression(t, "base * power ** 2 ** scale")
	multiply := requireType[*ast.BinaryExpression](t, expr)
	if multiply.Operator != ast.OperatorMul {
		t.Fatalf("expected * at root, got %q", multiply.Operator)
	}
	power := requireType[*ast.BinaryExpression](t, multiply.Right)
	if power.Operator != ast.OperatorPow {
		t.Fatalf("expected exponentiation on right side, got %q", power.Operator)
	}
	nested := requireType[*ast.BinaryExpression](t, power.Right)
	if nested.Operator != ast.OperatorPow {
		t.Fatalf("expected right-associative exponentiation, got %q", nested.Operator)
	}
}

func TestParserExpressionConditionalsAndComma(t *testing.T) {
	expr := parseExpression(t, "first, ok ? value ?? fallback : other")

	comma := requireType[*ast.CommaExpression](t, expr)
	if ident := requireType[*ast.Identifier](t, comma.Left); ident.Name != "first" {
		t.Fatalf("expected first expression identifier, got %q", ident.Name)
	}
	conditional := requireType[*ast.ConditionalExpression](t, comma.Right)
	if ident := requireType[*ast.Identifier](t, conditional.Condition); ident.Name != "ok" {
		t.Fatalf("expected conditional test identifier, got %q", ident.Name)
	}
	requireType[*ast.NullishCoalesceExpression](t, conditional.Consequent)
}

func TestParserExpressionUnaryAndInstanceof(t *testing.T) {
	expr := parseExpression(t, "typeof value instanceof Type || !ready")

	logical := requireType[*ast.LogicalExpression](t, expr)
	requireType[*ast.InstanceofExpression](t, logical.Left)
	unary := requireType[*ast.UnaryExpression](t, logical.Right)
	if unary.Operator != ast.OperatorNot {
		t.Fatalf("expected ! unary operator, got %q", unary.Operator)
	}
}

func TestParserExpressionUnaryPlus(t *testing.T) {
	expr := parseExpression(t, "+value")
	unary := requireType[*ast.UnaryExpression](t, expr)
	if unary.Operator != ast.OperatorPositive {
		t.Fatalf("expected unary + operator, got %q", unary.Operator)
	}
	requireType[*ast.Identifier](t, unary.Right)
}

func TestParserExpressionVoidOperator(t *testing.T) {
	expr := parseExpression(t, "void value")
	unary := requireType[*ast.UnaryExpression](t, expr)
	if unary.Operator != ast.OperatorVoid {
		t.Fatalf("expected void operator, got %q", unary.Operator)
	}
	requireType[*ast.Identifier](t, unary.Right)
}

func TestParserExpressionTemplateLiteral(t *testing.T) {
	expr := parseExpression(t, "`hello ${name}`")
	template := requireType[*ast.TemplateLiteral](t, expr)
	if template.Value != "hello ${name}" {
		t.Fatalf("unexpected template literal value %q", template.Value)
	}
}

func TestParserThisExpression(t *testing.T) {
	expr := parseExpression(t, "this.value")
	member := requireType[*ast.MemberExpression](t, expr)
	requireType[*ast.ThisExpression](t, member.Target)
	if member.Property != "value" {
		t.Fatalf("expected property value, got %q", member.Property)
	}
}

func TestParserKeywordMemberExpression(t *testing.T) {
	expr := parseExpression(t, "item.default")
	member := requireType[*ast.MemberExpression](t, expr)
	if member.Property != "default" {
		t.Fatalf("expected property default, got %q", member.Property)
	}
}

func parseExpression(t *testing.T, source string) ast.Expression {
	t.Helper()
	expr, err := parser.New(lexer.New(source)).ParseExpression()
	if err != nil {
		t.Fatalf("ParseExpression(%q) returned error: %v", source, err)
	}
	return expr
}

func requireType[T any](t *testing.T, value any) T {
	t.Helper()
	typed, ok := value.(T)
	if !ok {
		t.Fatalf("expected %T to be requested type", value)
	}
	return typed
}
