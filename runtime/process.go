package runtime

type ProcessCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func ProcessCapabilities() []ProcessCapability {
	return []ProcessCapability{
		{Name: "argv", RuntimeSymbol: "jayess_process_argv", Kind: "property"},
		{Name: "env", RuntimeSymbol: "jayess_process_env", Kind: "property"},
		{Name: "cwd", RuntimeSymbol: "jayess_process_cwd", Kind: "function"},
		{Name: "exit", RuntimeSymbol: "jayess_process_exit", Kind: "function"},
		{Name: "stdin", RuntimeSymbol: "jayess_process_stdin", Kind: "property"},
		{Name: "stdout", RuntimeSymbol: "jayess_process_stdout", Kind: "property"},
		{Name: "stderr", RuntimeSymbol: "jayess_process_stderr", Kind: "property"},
		{Name: "pid", RuntimeSymbol: "jayess_process_pid", Kind: "property"},
		{Name: "platform", RuntimeSymbol: "jayess_process_platform", Kind: "property"},
		{Name: "hrtime", RuntimeSymbol: "jayess_process_hrtime", Kind: "function"},
		{Name: "on", RuntimeSymbol: "jayess_process_on", Kind: "function"},
	}
}

func HasProcessCapability(name string) bool {
	for _, capability := range ProcessCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
