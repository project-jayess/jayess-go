package lowering

import "jayess-go/ast"

func evaluateTemplateString(template *ast.TemplateLiteral, bindings returnScope) (string, bool) {
	if template == nil {
		return "", false
	}
	runes := []rune(template.Value)
	result := make([]rune, 0, len(runes))
	expressionIndex := 0
	for index := 0; index < len(runes); index++ {
		if index+1 >= len(runes) || runes[index] != '$' || runes[index+1] != '{' {
			result = append(result, runes[index])
			continue
		}
		if expressionIndex >= len(template.Expressions) {
			return "", false
		}
		end := findTemplateStringExpressionEnd(runes, index+2)
		if end < 0 {
			return "", false
		}
		value, ok := evaluateStringCoercion(template.Expressions[expressionIndex], bindings)
		if !ok {
			return "", false
		}
		result = append(result, []rune(value)...)
		expressionIndex++
		index = end
	}
	if expressionIndex != len(template.Expressions) {
		return "", false
	}
	return string(result), true
}

func findTemplateStringExpressionEnd(runes []rune, start int) int {
	depth := 1
	for index := start; index < len(runes); index++ {
		switch runes[index] {
		case '\'', '"':
			index = skipQuotedTemplateStringExpression(runes, index, runes[index])
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

func skipQuotedTemplateStringExpression(runes []rune, start int, quote rune) int {
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
