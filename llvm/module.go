package llvm

import (
	"fmt"
	"strings"
)

type Module struct {
	name         string
	targetTriple string
	functions    []Function
}

func NewModule(name string) *Module {
	return &Module{name: name}
}

func (module *Module) SetTargetTriple(triple string) {
	module.targetTriple = triple
}

func (module *Module) AddFunction(name string, returnType Type) (*Function, error) {
	if name == "" {
		return nil, fmt.Errorf("LLVM function name must not be empty")
	}
	if returnType.ir == "" {
		return nil, fmt.Errorf("LLVM function %q must declare a return type", name)
	}
	function := Function{name: name, returnType: returnType}
	module.functions = append(module.functions, function)
	return &module.functions[len(module.functions)-1], nil
}

func (module *Module) String() string {
	var builder strings.Builder
	if module.targetTriple != "" {
		builder.WriteString("target triple = ")
		builder.WriteString(quoteString(module.targetTriple))
		builder.WriteString("\n\n")
	}
	for index := range module.functions {
		if index > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(module.functions[index].String())
		builder.WriteString("\n")
	}
	return builder.String()
}

func quoteString(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}
