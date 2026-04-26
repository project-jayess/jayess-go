package typesys

import (
	"fmt"
	"strings"
	"unicode"
)

type Kind int

const (
	KindAny Kind = iota
	KindSimple
	KindLiteral
	KindUnion
	KindIntersection
	KindTuple
	KindObject
	KindFunction
	KindApplication
)

type Expr struct {
	Kind            Kind
	Name            string
	Elements        []*Expr
	Properties      []Property
	IndexSignatures []IndexSignature
	Params          []*Expr
	Return          *Expr
	TypeArgs        []*Expr
}

type Property struct {
	Name     string
	Optional bool
	Readonly bool
	Type     *Expr
}

type IndexSignature struct {
	KeyName   string
	KeyType   *Expr
	ValueType *Expr
	Readonly  bool
}

type parser struct {
	text string
	pos  int
}

func Normalize(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	expr, err := Parse(name)
	if err != nil {
		return name
	}
	if expr.Kind == KindAny {
		return ""
	}
	return expr.String()
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

func RewriteAliases(name string, rewriteSimple func(string) (string, error)) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", nil
	}
	expr, err := Parse(name)
	if err != nil {
		return "", err
	}
	rewritten, err := rewriteExprAliases(expr, rewriteSimple)
	if err != nil {
		return "", err
	}
	if rewritten.Kind == KindAny {
		return "", nil
	}
	return rewritten.String(), nil
}

func rewriteExprAliases(expr *Expr, rewriteSimple func(string) (string, error)) (*Expr, error) {
	switch expr.Kind {
	case KindAny:
		return &Expr{Kind: KindAny}, nil
	case KindSimple:
		rewritten, err := rewriteSimple(expr.Name)
		if err != nil {
			return nil, err
		}
		if rewritten == "" {
			return &Expr{Kind: KindAny}, nil
		}
		return Parse(rewritten)
	case KindLiteral:
		return &Expr{Kind: KindLiteral, Name: expr.Name}, nil
	case KindUnion:
		out := &Expr{Kind: KindUnion, Elements: make([]*Expr, len(expr.Elements))}
		for i, element := range expr.Elements {
			rewritten, err := rewriteExprAliases(element, rewriteSimple)
			if err != nil {
				return nil, err
			}
			out.Elements[i] = rewritten
		}
		return out, nil
	case KindIntersection:
		out := &Expr{Kind: KindIntersection, Elements: make([]*Expr, len(expr.Elements))}
		for i, element := range expr.Elements {
			rewritten, err := rewriteExprAliases(element, rewriteSimple)
			if err != nil {
				return nil, err
			}
			out.Elements[i] = rewritten
		}
		return out, nil
	case KindTuple:
		out := &Expr{Kind: KindTuple, Elements: make([]*Expr, len(expr.Elements))}
		for i, element := range expr.Elements {
			rewritten, err := rewriteExprAliases(element, rewriteSimple)
			if err != nil {
				return nil, err
			}
			out.Elements[i] = rewritten
		}
		return out, nil
	case KindObject:
		out := &Expr{Kind: KindObject, Properties: make([]Property, len(expr.Properties)), IndexSignatures: make([]IndexSignature, len(expr.IndexSignatures))}
		for i, property := range expr.Properties {
			rewritten, err := rewriteExprAliases(property.Type, rewriteSimple)
			if err != nil {
				return nil, err
			}
			out.Properties[i] = property
			out.Properties[i].Type = rewritten
		}
		for i, signature := range expr.IndexSignatures {
			keyType, err := rewriteExprAliases(signature.KeyType, rewriteSimple)
			if err != nil {
				return nil, err
			}
			valueType, err := rewriteExprAliases(signature.ValueType, rewriteSimple)
			if err != nil {
				return nil, err
			}
			out.IndexSignatures[i] = signature
			out.IndexSignatures[i].KeyType = keyType
			out.IndexSignatures[i].ValueType = valueType
		}
		return out, nil
	case KindFunction:
		out := &Expr{Kind: KindFunction, Params: make([]*Expr, len(expr.Params))}
		for i, param := range expr.Params {
			rewritten, err := rewriteExprAliases(param, rewriteSimple)
			if err != nil {
				return nil, err
			}
			out.Params[i] = rewritten
		}
		rewrittenReturn, err := rewriteExprAliases(expr.Return, rewriteSimple)
		if err != nil {
			return nil, err
		}
		out.Return = rewrittenReturn
		return out, nil
	case KindApplication:
		out := &Expr{Kind: KindApplication, Name: expr.Name, TypeArgs: make([]*Expr, len(expr.TypeArgs))}
		for i, arg := range expr.TypeArgs {
			rewritten, err := rewriteExprAliases(arg, rewriteSimple)
			if err != nil {
				return nil, err
			}
			out.TypeArgs[i] = rewritten
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported type expression kind %d", expr.Kind)
	}
}

func (e *Expr) String() string {
	if e == nil {
		return ""
	}
	switch e.Kind {
	case KindAny:
		return ""
	case KindSimple:
		switch e.Name {
		case "bool":
			return "boolean"
		case "any", "dynamic":
			return ""
		default:
			return e.Name
		}
	case KindLiteral:
		return e.Name
	case KindUnion:
		parts := make([]string, len(e.Elements))
		for i, element := range e.Elements {
			parts[i] = element.String()
		}
		return strings.Join(parts, "|")
	case KindIntersection:
		parts := make([]string, len(e.Elements))
		for i, element := range e.Elements {
			parts[i] = element.String()
		}
		return strings.Join(parts, "&")
	case KindTuple:
		parts := make([]string, len(e.Elements))
		for i, element := range e.Elements {
			parts[i] = element.String()
		}
		return "[" + strings.Join(parts, ",") + "]"
	case KindObject:
		parts := make([]string, 0, len(e.Properties)+len(e.IndexSignatures))
		for _, property := range e.Properties {
			prefix := ""
			if property.Readonly {
				prefix = "readonly "
			}
			suffix := ""
			if property.Optional {
				suffix = "?"
			}
			parts = append(parts, prefix+property.Name+suffix+":"+property.Type.String())
		}
		for _, signature := range e.IndexSignatures {
			prefix := ""
			if signature.Readonly {
				prefix = "readonly "
			}
			parts = append(parts, prefix+"["+signature.KeyName+":"+signature.KeyType.String()+"]:"+signature.ValueType.String())
		}
		return "{" + strings.Join(parts, ",") + "}"
	case KindFunction:
		parts := make([]string, len(e.Params))
		for i, param := range e.Params {
			parts[i] = param.String()
		}
		return "(" + strings.Join(parts, ",") + ")=>" + e.Return.String()
	case KindApplication:
		parts := make([]string, len(e.TypeArgs))
		for i, arg := range e.TypeArgs {
			parts[i] = arg.String()
		}
		return e.Name + "<" + strings.Join(parts, ",") + ">"
	default:
		return ""
	}
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
