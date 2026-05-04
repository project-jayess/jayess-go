package llvmbackend

import "strings"

type Module struct {
	Name         string
	Target       TargetConfig
	Functions    []Function
	Declarations []Declaration
}

type Function struct {
	Name       string
	ReturnType string
	Body       []string
}

type Declaration struct {
	Name   string
	IRType string
}

func EmitLLVMIR(module Module) string {
	var b strings.Builder
	b.WriteString("; ModuleID = '")
	b.WriteString(module.Name)
	b.WriteString("'\n")
	if module.Target.Triple != "" {
		b.WriteString("target triple = \"")
		b.WriteString(module.Target.Triple)
		b.WriteString("\"\n")
	}
	for _, declaration := range module.Declarations {
		b.WriteString("\ndeclare ")
		b.WriteString(declaration.IRType)
		b.WriteString(" @")
		b.WriteString(declaration.Name)
		b.WriteString("\n")
	}
	for _, fn := range module.Functions {
		b.WriteString("\ndefine ")
		b.WriteString(fn.ReturnType)
		b.WriteString(" @")
		b.WriteString(fn.Name)
		b.WriteString("() {\n")
		for _, line := range fn.Body {
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("}\n")
	}
	return b.String()
}
