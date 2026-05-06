package llvmbackend

import "strings"

type Module struct {
	Name         string
	Target       TargetConfig
	Globals      []Global
	Functions    []Function
	Declarations []Declaration
}

type Global struct {
	Name   string
	IRType string
	Value  string
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
	if moduleUsesRuntimeValue(module) {
		b.WriteString("\n")
		b.WriteString(runtimeValueIRType)
		b.WriteString(" = type { i64, i64 }\n")
	}
	for _, global := range module.Globals {
		b.WriteString("\n@")
		b.WriteString(global.Name)
		b.WriteString(" = private unnamed_addr constant ")
		b.WriteString(global.IRType)
		b.WriteString(" ")
		b.WriteString(global.Value)
		b.WriteString("\n")
	}
	for _, declaration := range module.Declarations {
		b.WriteString("\ndeclare ")
		b.WriteString(formatDeclarationIRType(declaration))
		b.WriteString(" @")
		b.WriteString(declaration.Name)
		if args, ok := declarationArgumentList(declaration.IRType); ok {
			b.WriteString(args)
		}
		b.WriteString("\n")
		if _, ok := declarationArgumentList(declaration.IRType); ok {
			b.WriteString("; legacy declare ")
			b.WriteString(declaration.IRType)
			b.WriteString(" @")
			b.WriteString(declaration.Name)
			b.WriteString("\n")
		}
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

func formatDeclarationIRType(declaration Declaration) string {
	if index := strings.Index(declaration.IRType, " ("); index >= 0 {
		return declaration.IRType[:index]
	}
	return declaration.IRType
}

func declarationArgumentList(irType string) (string, bool) {
	if index := strings.Index(irType, " ("); index >= 0 {
		return irType[index+1:], true
	}
	return "", false
}

func moduleUsesRuntimeValue(module Module) bool {
	for _, declaration := range module.Declarations {
		if strings.Contains(declaration.IRType, runtimeValueIRType) {
			return true
		}
	}
	for _, fn := range module.Functions {
		if strings.Contains(fn.ReturnType, runtimeValueIRType) {
			return true
		}
		for _, line := range fn.Body {
			if strings.Contains(line, runtimeValueIRType) {
				return true
			}
		}
	}
	return false
}
