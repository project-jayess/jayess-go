package tooling

type Option struct {
	Name        string
	Description string
}

func Options() []Option {
	return []Option{
		{Name: "target", Description: "select compilation target"},
		{Name: "output", Description: "select output file"},
		{Name: "executable", Description: "select an executable to package for dist output"},
		{Name: "emit", Description: "select emitted artifact kind"},
		{Name: "debug-info", Description: "include DWARF debug information"},
	}
}

func HasOption(name string) bool {
	for _, option := range Options() {
		if option.Name == name {
			return true
		}
	}
	return false
}
