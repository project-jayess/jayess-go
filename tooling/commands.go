package tooling

type Command struct {
	Name        string
	Description string
}

func Commands() []Command {
	return []Command{
		{Name: "compile", Description: "compile a .js input file"},
		{Name: "package", Description: "package a compiled Jayess executable with runtime assets"},
		{Name: "run", Description: "compile and run a .js input file"},
		{Name: "init", Description: "initialize a Jayess package directory"},
		{Name: "test", Description: "discover and run Jayess test files"},
	}
}

func HasCommand(name string) bool {
	for _, command := range Commands() {
		if command.Name == name {
			return true
		}
	}
	return false
}
