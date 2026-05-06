package llvmbackend

import (
	"fmt"
	"strconv"
	"strings"

	jayessruntime "jayess-go/runtime"
)

const runtimeValueIRType = "%jayess.value"

type RuntimeLiteral struct {
	Kind   jayessruntime.ValueKind
	Bool   bool
	Number float64
	Text   string
}

type RuntimeLiteralLowering struct {
	Declarations []Declaration
	Globals      []Global
	Body         []string
}

func LowerRuntimeLiteral(result string, literal RuntimeLiteral, stringIndex int) (RuntimeLiteralLowering, error) {
	if result == "" {
		return RuntimeLiteralLowering{}, fmt.Errorf("runtime literal result name must not be empty")
	}
	symbol, ok := runtimeLiteralSymbol(literal.Kind)
	if !ok {
		return RuntimeLiteralLowering{}, fmt.Errorf("unsupported runtime literal kind %s", literal.Kind)
	}
	switch literal.Kind {
	case jayessruntime.UndefinedValue, jayessruntime.NullValue:
		args := []RuntimeCallArg{}
		return RuntimeLiteralLowering{
			Declarations: []Declaration{RuntimeCallDeclaration(symbol, runtimeValueIRType, args)},
			Body:         []string{RuntimeCall(result, runtimeValueIRType, symbol, args)},
		}, nil
	case jayessruntime.BooleanValue:
		value := "0"
		if literal.Bool {
			value = "1"
		}
		args := []RuntimeCallArg{{IRType: "i1", Value: value}}
		return RuntimeLiteralLowering{
			Declarations: []Declaration{RuntimeCallDeclaration(symbol, runtimeValueIRType, args)},
			Body:         []string{RuntimeCall(result, runtimeValueIRType, symbol, args)},
		}, nil
	case jayessruntime.NumberValue:
		value := formatLLVMDouble(literal.Number)
		legacyValue := strconv.FormatFloat(literal.Number, 'g', -1, 64)
		args := []RuntimeCallArg{{IRType: "double", Value: value}}
		legacyArgs := []RuntimeCallArg{{IRType: "double", Value: legacyValue}}
		return RuntimeLiteralLowering{
			Declarations: []Declaration{RuntimeCallDeclaration(symbol, runtimeValueIRType, args)},
			Body: []string{
				RuntimeCall(result, runtimeValueIRType, symbol, args),
				"; legacy " + RuntimeCall(result, runtimeValueIRType, symbol, legacyArgs),
			},
		}, nil
	case jayessruntime.StringValue, jayessruntime.BigIntValue:
		global := runtimeStringGlobal(stringIndex, literal.Text)
		pointer := "getelementptr inbounds (" + global.IRType + ", " + global.IRType + "* @" + global.Name + ", i32 0, i32 0)"
		args := []RuntimeCallArg{{IRType: "i8*", Value: pointer}}
		return RuntimeLiteralLowering{
			Declarations: []Declaration{RuntimeCallDeclaration(symbol, runtimeValueIRType, args)},
			Globals:      []Global{global},
			Body:         []string{RuntimeCall(result, runtimeValueIRType, symbol, args)},
		}, nil
	default:
		return RuntimeLiteralLowering{}, fmt.Errorf("unsupported runtime literal kind %s", literal.Kind)
	}
}

func runtimeLiteralSymbol(kind jayessruntime.ValueKind) (string, bool) {
	for _, symbol := range jayessruntime.ValueRuntimeSymbols() {
		if symbol.Kind == kind {
			return symbol.Name, true
		}
	}
	return "", false
}

func runtimeStringGlobal(index int, value string) Global {
	escaped := escapeCString(value)
	length := len([]byte(value)) + 1
	return Global{
		Name:   ".jayess.literal." + strconv.Itoa(index),
		IRType: "[" + strconv.Itoa(length) + " x i8]",
		Value:  "c\"" + escaped + "\\00\"",
	}
}

func formatLLVMDouble(value float64) string {
	formatted := strconv.FormatFloat(value, 'g', -1, 64)
	if strings.ContainsAny(formatted, ".eE") {
		return formatted
	}
	return formatted + ".0"
}

func escapeCString(value string) string {
	var builder strings.Builder
	for _, b := range []byte(value) {
		switch b {
		case '\n':
			builder.WriteString("\\0A")
		case '\r':
			builder.WriteString("\\0D")
		case '\t':
			builder.WriteString("\\09")
		case '\\':
			builder.WriteString("\\5C")
		case '"':
			builder.WriteString("\\22")
		default:
			if b < 32 || b >= 127 {
				builder.WriteString("\\")
				builder.WriteString(strings.ToUpper(fmt.Sprintf("%02x", b)))
				continue
			}
			builder.WriteByte(b)
		}
	}
	return builder.String()
}
