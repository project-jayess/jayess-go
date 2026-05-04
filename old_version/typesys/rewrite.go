package typesys

import (
	"fmt"
	"strings"
)

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
