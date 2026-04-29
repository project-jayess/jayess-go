package semantic

import (
	"fmt"
	"strings"

	"jayess-go/ast"
	"jayess-go/typesys"
)

func cloneSymbols(input map[string]symbol) map[string]symbol {
	out := make(map[string]symbol, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func hasRestParam(params []ast.Parameter) bool {
	return len(params) > 0 && params[len(params)-1].Rest
}

func minRequiredParams(params []ast.Parameter) int {
	count := 0
	for _, param := range params {
		if param.Rest {
			return count
		}
		if param.Default == nil {
			count++
		}
	}
	return count
}

func validateParameterList(params []ast.Parameter) error {
	seenDefault := false
	for i, param := range params {
		if param.TypeAnnotation != "" && !isSupportedTypeAnnotation(param.TypeAnnotation) {
			return fmt.Errorf("unsupported type annotation %s", param.TypeAnnotation)
		}
		if param.Rest && i != len(params)-1 {
			return fmt.Errorf("rest parameter must be last")
		}
		if param.Rest && param.Default != nil {
			return fmt.Errorf("rest parameter cannot have a default value")
		}
		if param.Default != nil {
			seenDefault = true
		} else if seenDefault && !param.Rest {
			return fmt.Errorf("parameters without defaults cannot follow parameters with defaults")
		}
	}
	return nil
}

func validateVariableAnnotation(decl *ast.VariableDecl) error {
	if decl.TypeAnnotation == "" {
		return nil
	}
	if !isSupportedTypeAnnotation(decl.TypeAnnotation) {
		return errorAt(decl, "variable %s has unsupported type annotation %s", decl.Name, decl.TypeAnnotation)
	}
	if normalizeTypeAnnotation(decl.TypeAnnotation) == "void" {
		return errorAt(decl, "variable %s cannot be annotated as void", decl.Name)
	}
	return nil
}

func parameterTypes(params []ast.Parameter) []string {
	out := make([]string, len(params))
	for i, param := range params {
		out[i] = normalizeTypeAnnotation(param.TypeAnnotation)
	}
	return out
}

func normalizeTypeAnnotation(name string) string {
	return typesys.Normalize(name)
}

func isAssignableTo(expected string, actual string) bool {
	expected = normalizeTypeAnnotation(expected)
	actual = normalizeTypeAnnotation(actual)
	if expected == "" || actual == "dynamic" {
		return true
	}
	if expected == "unknown" {
		return true
	}
	if actual == "unknown" {
		return expected == "unknown"
	}
	if expected == "array" && actual == "args_array" {
		return true
	}
	if expected == "void" {
		return actual == "void" || actual == "undefined"
	}
	if expected == "never" {
		return actual == "never"
	}
	if structured, err := typesys.Parse(expected); err == nil {
		switch structured.Kind {
		case typesys.KindLiteral:
			switch structured.Name {
			case "true", "false":
				return actual == "boolean" || actual == structured.Name
			default:
				if strings.HasPrefix(structured.Name, "\"") {
					return actual == "string" || actual == structured.Name
				}
				return actual == "number" || actual == structured.Name
			}
		case typesys.KindUnion:
			for _, member := range structured.Elements {
				if isAssignableTo(member.String(), actual) {
					return true
				}
			}
			return false
		case typesys.KindIntersection:
			for _, member := range structured.Elements {
				if !isAssignableTo(member.String(), actual) {
					return false
				}
			}
			return true
		case typesys.KindTuple:
			return actual == "array" || actual == "args_array" || actual == expected
		case typesys.KindObject:
			return actual == "object" || actual == expected
		case typesys.KindFunction:
			return actual == "function" || actual == expected
		}
	}
	return expected == actual
}

func isSupportedTypeAnnotation(name string) bool {
	return typesys.IsSupported(name)
}
