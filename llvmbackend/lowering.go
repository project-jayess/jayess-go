package llvmbackend

type JayessProgram struct {
	Name                 string
	Target               TargetConfig
	EntryName            string
	ReturnCode           int
	ModuleInitialization ModuleInitializationPlan
}

func LowerJayessProgram(program JayessProgram) Module {
	entry := program.EntryName
	if entry == "" {
		entry = "main"
	}
	body := moduleInitializationCalls(program.ModuleInitialization)
	body = append(body, "ret i32 "+itoa(program.ReturnCode))
	return Module{
		Name:   program.Name,
		Target: program.Target,
		Functions: []Function{
			{
				Name:       entry,
				ReturnType: "i32",
				Body:       body,
			},
		},
		Declarations: moduleInitializationDeclarations(program.ModuleInitialization),
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits []byte
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
