package parser

import (
	"fmt"

	"jayess-go/ast"
	"jayess-go/lexer"
)

func (p *Parser) parseTemplateLiteral(token lexer.Token) (*ast.TemplateLiteral, error) {
	expressions, err := parseTemplateExpressions(token.Literal)
	if err != nil {
		return nil, errorAtToken(token, "%s", err)
	}
	return &ast.TemplateLiteral{BaseNode: baseFrom(token), Value: token.Literal, Expressions: expressions}, nil
}

func parseTemplateExpressions(value string) ([]ast.Expression, error) {
	runes := []rune(value)
	expressions := []ast.Expression{}
	for index := 0; index < len(runes)-1; index++ {
		if runes[index] != '$' || runes[index+1] != '{' {
			continue
		}
		start := index + 2
		end := findTemplateExpressionEnd(runes, start)
		if end < 0 {
			return nil, fmt.Errorf("unterminated template expression")
		}
		expr, err := New(lexer.New(string(runes[start:end]))).ParseExpression()
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expr)
		index = end
	}
	return expressions, nil
}

func findTemplateExpressionEnd(runes []rune, start int) int {
	depth := 1
	for index := start; index < len(runes); index++ {
		switch runes[index] {
		case '\'', '"':
			index = skipQuotedTemplateExpressionPart(runes, index, runes[index])
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func skipQuotedTemplateExpressionPart(runes []rune, start int, quote rune) int {
	for index := start + 1; index < len(runes); index++ {
		if runes[index] == '\\' {
			index++
			continue
		}
		if runes[index] == quote {
			return index
		}
	}
	return len(runes) - 1
}
