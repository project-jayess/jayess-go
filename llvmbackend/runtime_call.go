package llvmbackend

import "strings"

type RuntimeCallArg struct {
	IRType string
	Value  string
}

func RuntimeCallDeclaration(symbol string, resultType string, args []RuntimeCallArg) Declaration {
	argTypes := make([]string, 0, len(args))
	for _, arg := range args {
		argTypes = append(argTypes, arg.IRType)
	}
	return Declaration{Name: symbol, IRType: resultType + " (" + strings.Join(argTypes, ", ") + ")"}
}

func RuntimeCall(result string, resultType string, symbol string, args []RuntimeCallArg) string {
	values := make([]string, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.IRType+" "+arg.Value)
	}
	return result + " = call " + resultType + " @" + symbol + "(" + strings.Join(values, ", ") + ")"
}

func RuntimeVoidCall(symbol string, args []RuntimeCallArg) string {
	values := make([]string, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.IRType+" "+arg.Value)
	}
	return "call void @" + symbol + "(" + strings.Join(values, ", ") + ")"
}
