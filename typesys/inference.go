package typesys

import "strings"

type InferenceSource string

const (
	InferFromNumberLiteral    InferenceSource = "number-literal"
	InferFromStringLiteral    InferenceSource = "string-literal"
	InferFromBooleanLiteral   InferenceSource = "boolean-literal"
	InferFromNullLiteral      InferenceSource = "null-literal"
	InferFromUndefinedLiteral InferenceSource = "undefined-literal"
	InferFromArrayLiteral     InferenceSource = "array-literal"
	InferFromObjectLiteral    InferenceSource = "object-literal"
	InferFromUnknown          InferenceSource = "unknown"
)

type Inference struct {
	Source    InferenceSource
	TypeName  string
	Confident bool
}

func InferLocalLiteral(source string) Inference {
	trimmed := strings.TrimSpace(source)
	switch {
	case trimmed == "true" || trimmed == "false":
		return Inference{Source: InferFromBooleanLiteral, TypeName: "boolean", Confident: true}
	case trimmed == "null":
		return Inference{Source: InferFromNullLiteral, TypeName: "null", Confident: true}
	case trimmed == "undefined":
		return Inference{Source: InferFromUndefinedLiteral, TypeName: "undefined", Confident: true}
	case strings.HasPrefix(trimmed, "\"") || strings.HasPrefix(trimmed, "'") || strings.HasPrefix(trimmed, "`"):
		return Inference{Source: InferFromStringLiteral, TypeName: "string", Confident: true}
	case strings.HasPrefix(trimmed, "["):
		return Inference{Source: InferFromArrayLiteral, TypeName: "array", Confident: true}
	case strings.HasPrefix(trimmed, "{"):
		return Inference{Source: InferFromObjectLiteral, TypeName: "object", Confident: true}
	case isNumberLiteral(trimmed):
		return Inference{Source: InferFromNumberLiteral, TypeName: "number", Confident: true}
	default:
		return Inference{Source: InferFromUnknown, TypeName: "unknown"}
	}
}

func isNumberLiteral(value string) bool {
	if value == "" {
		return false
	}
	digits := 0
	dots := 0
	for index, r := range value {
		if r == '-' && index == 0 {
			continue
		}
		if r == '.' {
			dots++
			if dots > 1 {
				return false
			}
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
		digits++
	}
	return digits > 0
}
