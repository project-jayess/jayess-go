package llvmbackend

import "strings"

type ToolchainCommand struct {
	Step    ToolchainStep
	Program string
	Args    []string
}

func (command ToolchainCommand) String() string {
	parts := []string{command.Program}
	for _, arg := range command.Args {
		parts = append(parts, quoteCommandArg(arg))
	}
	return strings.Join(parts, " ")
}

func quoteCommandArg(arg string) string {
	if arg == "" {
		return `""`
	}
	if !strings.ContainsAny(arg, " \t\n\"") {
		return arg
	}
	return `"` + strings.ReplaceAll(arg, `"`, `\"`) + `"`
}
