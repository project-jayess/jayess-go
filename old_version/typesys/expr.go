package typesys

import "strings"

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
