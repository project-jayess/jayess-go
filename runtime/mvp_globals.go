package runtime

type MVPGlobalImplementation struct {
	Name          string
	RuntimeSymbol string
	Methods       []string
}

func MVPGlobalImplementations() []MVPGlobalImplementation {
	return []MVPGlobalImplementation{
		{Name: "console", RuntimeSymbol: "jayess_console", Methods: []string{"log"}},
		{Name: "print", RuntimeSymbol: "jayess_print"},
		{Name: "sleep", RuntimeSymbol: "jayess_sleep"},
		{Name: "readLine", RuntimeSymbol: "jayess_read_line"},
		{Name: "readKey", RuntimeSymbol: "jayess_read_key"},
	}
}

func HasMVPGlobalImplementation(name string) bool {
	for _, implementation := range MVPGlobalImplementations() {
		if implementation.Name == name {
			return implementation.RuntimeSymbol != ""
		}
	}
	return false
}
