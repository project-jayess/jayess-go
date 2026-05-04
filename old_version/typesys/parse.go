package typesys

import (
	"fmt"
	"strings"
	"unicode"
)

type parser struct {
	text string
	pos  int
}

func Parse(name string) (*Expr, error) {
	p := &parser{text: strings.TrimSpace(name)}
	expr, err := p.parseType()
	if err != nil {
		return nil, err
	}
	p.skipSpaces()
	if p.pos != len(p.text) {
		return nil, fmt.Errorf("unexpected trailing type text %q", p.text[p.pos:])
	}
	return expr, nil
}

func IsSupported(name string) bool {
	_, err := Parse(name)
	return err == nil
}

func (p *parser) parseType() (*Expr, error) {
	return p.parseUnion()
}

func (p *parser) parseUnion() (*Expr, error) {
	left, err := p.parseIntersection()
	if err != nil {
		return nil, err
	}
	p.skipSpaces()
	if !p.consume("|") {
		return left, nil
	}
	elements := []*Expr{left}
	for {
		right, err := p.parseIntersection()
		if err != nil {
			return nil, err
		}
		elements = append(elements, right)
		p.skipSpaces()
		if !p.consume("|") {
			break
		}
	}
	return &Expr{Kind: KindUnion, Elements: elements}, nil
}

func (p *parser) parseIntersection() (*Expr, error) {
	left, err := p.parsePrimaryType()
	if err != nil {
		return nil, err
	}
	p.skipSpaces()
	if !p.consume("&") {
		return left, nil
	}
	elements := []*Expr{left}
	for {
		right, err := p.parsePrimaryType()
		if err != nil {
			return nil, err
		}
		elements = append(elements, right)
		p.skipSpaces()
		if !p.consume("&") {
			break
		}
	}
	return &Expr{Kind: KindIntersection, Elements: elements}, nil
}

func (p *parser) parsePrimaryType() (*Expr, error) {
	p.skipSpaces()
	if p.pos >= len(p.text) {
		return &Expr{Kind: KindAny}, nil
	}
	switch p.text[p.pos] {
	case '[':
		return p.parseTuple()
	case '{':
		return p.parseObject()
	case '(':
		return p.parseCallableOrGrouped()
	default:
		return p.parseSimple()
	}
}

func (p *parser) parseTuple() (*Expr, error) {
	p.pos++
	expr := &Expr{Kind: KindTuple}
	p.skipSpaces()
	if p.consume("]") {
		return expr, nil
	}
	for {
		element, err := p.parseType()
		if err != nil {
			return nil, err
		}
		expr.Elements = append(expr.Elements, element)
		p.skipSpaces()
		if p.consume("]") {
			return expr, nil
		}
		if !p.consume(",") {
			return nil, fmt.Errorf("expected ',' or ']' in tuple type")
		}
	}
}

func (p *parser) parseObject() (*Expr, error) {
	p.pos++
	expr := &Expr{Kind: KindObject}
	p.skipSpaces()
	if p.consume("}") {
		return expr, nil
	}
	for {
		readonly := false
		if p.hasWord("readonly") {
			p.consumeWord("readonly")
			readonly = true
			p.skipSpaces()
		}
		if p.consume("[") {
			keyName, err := p.parseIdentifier()
			if err != nil {
				return nil, err
			}
			if !p.consume(":") {
				return nil, fmt.Errorf("expected ':' in index signature")
			}
			keyType, err := p.parseType()
			if err != nil {
				return nil, err
			}
			if !p.consume("]") {
				return nil, fmt.Errorf("expected ']' in index signature")
			}
			if !p.consume(":") {
				return nil, fmt.Errorf("expected ':' after index signature")
			}
			valueType, err := p.parseType()
			if err != nil {
				return nil, err
			}
			expr.IndexSignatures = append(expr.IndexSignatures, IndexSignature{
				KeyName:   keyName,
				KeyType:   keyType,
				ValueType: valueType,
				Readonly:  readonly,
			})
			p.skipSpaces()
			if p.consume("}") {
				return expr, nil
			}
			if !p.consume(",") && !p.consume(";") {
				return nil, fmt.Errorf("expected ',' ';' or '}' in object type")
			}
			p.skipSpaces()
			if p.consume("}") {
				return expr, nil
			}
			continue
		}
		property := Property{Readonly: readonly}
		name, err := p.parseIdentifier()
		if err != nil {
			return nil, err
		}
		property.Name = name
		p.skipSpaces()
		if p.consume("?") {
			property.Optional = true
		}
		if !p.consume(":") {
			return nil, fmt.Errorf("expected ':' in object type")
		}
		value, err := p.parseType()
		if err != nil {
			return nil, err
		}
		property.Type = value
		expr.Properties = append(expr.Properties, property)
		p.skipSpaces()
		if p.consume("}") {
			return expr, nil
		}
		if !p.consume(",") && !p.consume(";") {
			return nil, fmt.Errorf("expected ',' ';' or '}' in object type")
		}
		p.skipSpaces()
		if p.consume("}") {
			return expr, nil
		}
	}
}

func (p *parser) parseCallableOrGrouped() (*Expr, error) {
	p.pos++
	params := []*Expr{}
	p.skipSpaces()
	if !p.consume(")") {
		for {
			p.skipSpaces()
			if name, ok := p.tryIdentifier(); ok {
				saved := p.pos
				p.skipSpaces()
				if p.consume(":") {
					paramType, err := p.parseType()
					if err != nil {
						return nil, err
					}
					params = append(params, paramType)
				} else {
					p.pos = saved
					params = append(params, &Expr{Kind: KindSimple, Name: name})
				}
			} else {
				paramType, err := p.parseType()
				if err != nil {
					return nil, err
				}
				params = append(params, paramType)
			}
			p.skipSpaces()
			if p.consume(")") {
				break
			}
			if !p.consume(",") {
				return nil, fmt.Errorf("expected ',' or ')' in callable type")
			}
		}
	}
	p.skipSpaces()
	if p.consume("=>") {
		returnType, err := p.parseType()
		if err != nil {
			return nil, err
		}
		return &Expr{Kind: KindFunction, Params: params, Return: returnType}, nil
	}
	if len(params) == 1 {
		return params[0], nil
	}
	return nil, fmt.Errorf("grouped type must contain exactly one inner type")
}

func (p *parser) parseSimple() (*Expr, error) {
	p.skipSpaces()
	if p.pos < len(p.text) && p.text[p.pos] == '"' {
		return p.parseQuotedLiteral()
	}
	if p.pos < len(p.text) && unicode.IsDigit(rune(p.text[p.pos])) {
		return p.parseNumberLiteral()
	}
	name, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}
	switch name {
	case "any", "dynamic":
		return &Expr{Kind: KindAny}, nil
	case "true", "false":
		return &Expr{Kind: KindLiteral, Name: name}, nil
	default:
		p.skipSpaces()
		if p.consume("<") {
			args := []*Expr{}
			p.skipSpaces()
			if !p.consume(">") {
				for {
					arg, err := p.parseType()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					p.skipSpaces()
					if p.consume(">") {
						break
					}
					if !p.consume(",") {
						return nil, fmt.Errorf("expected ',' or '>' in generic type arguments")
					}
				}
			}
			return &Expr{Kind: KindApplication, Name: name, TypeArgs: args}, nil
		}
		return &Expr{Kind: KindSimple, Name: name}, nil
	}
}

func (p *parser) parseQuotedLiteral() (*Expr, error) {
	start := p.pos
	p.pos++
	for p.pos < len(p.text) {
		if p.text[p.pos] == '\\' {
			p.pos += 2
			continue
		}
		if p.text[p.pos] == '"' {
			p.pos++
			return &Expr{Kind: KindLiteral, Name: p.text[start:p.pos]}, nil
		}
		p.pos++
	}
	return nil, fmt.Errorf("unterminated string literal type")
}

func (p *parser) parseNumberLiteral() (*Expr, error) {
	start := p.pos
	for p.pos < len(p.text) {
		r := rune(p.text[p.pos])
		if !unicode.IsDigit(r) && r != '.' {
			break
		}
		p.pos++
	}
	return &Expr{Kind: KindLiteral, Name: p.text[start:p.pos]}, nil
}

func (p *parser) parseIdentifier() (string, error) {
	if ident, ok := p.tryIdentifier(); ok {
		return ident, nil
	}
	return "", fmt.Errorf("expected type identifier")
}

func (p *parser) tryIdentifier() (string, bool) {
	p.skipSpaces()
	if p.pos >= len(p.text) {
		return "", false
	}
	start := p.pos
	r := rune(p.text[p.pos])
	if !unicode.IsLetter(r) && r != '_' {
		return "", false
	}
	p.pos++
	for p.pos < len(p.text) {
		r = rune(p.text[p.pos])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			break
		}
		p.pos++
	}
	return p.text[start:p.pos], true
}

func (p *parser) skipSpaces() {
	for p.pos < len(p.text) && unicode.IsSpace(rune(p.text[p.pos])) {
		p.pos++
	}
}

func (p *parser) consume(text string) bool {
	p.skipSpaces()
	if strings.HasPrefix(p.text[p.pos:], text) {
		p.pos += len(text)
		return true
	}
	return false
}

func (p *parser) hasWord(word string) bool {
	p.skipSpaces()
	if !strings.HasPrefix(p.text[p.pos:], word) {
		return false
	}
	end := p.pos + len(word)
	if end < len(p.text) {
		r := rune(p.text[end])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return false
		}
	}
	return true
}

func (p *parser) consumeWord(word string) {
	if p.hasWord(word) {
		p.pos += len(word)
	}
}
