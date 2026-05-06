package runtime

import (
	"os"
	"strconv"
)

type TerminalInfo struct {
	IsTerminal    bool
	Columns       int
	Rows          int
	SupportsColor bool
}

type TerminalCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func TerminalCapabilities() []TerminalCapability {
	return []TerminalCapability{
		{Name: "isTTY", RuntimeSymbol: "jayess_terminal_is_tty", Kind: "function"},
		{Name: "size", RuntimeSymbol: "jayess_terminal_size", Kind: "function"},
		{Name: "supportsColor", RuntimeSymbol: "jayess_terminal_supports_color", Kind: "function"},
	}
}

func HasTerminalCapability(name string) bool {
	for _, capability := range TerminalCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}

func DetectTerminal(file *os.File, env map[string]string) TerminalInfo {
	if env == nil {
		env = osEnvironment()
	}
	info := TerminalInfo{
		IsTerminal:    isTerminalFile(file),
		Columns:       intFromEnv(env, "COLUMNS"),
		Rows:          intFromEnv(env, "LINES"),
		SupportsColor: supportsColor(env),
	}
	return info
}

func isTerminalFile(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func intFromEnv(env map[string]string, name string) int {
	value, ok := env[name]
	if !ok {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func supportsColor(env map[string]string) bool {
	if env["NO_COLOR"] != "" {
		return false
	}
	term := env["TERM"]
	return term != "" && term != "dumb"
}
