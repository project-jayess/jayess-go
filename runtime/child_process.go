package runtime

type ChildProcessCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func ChildProcessCapabilities() []ChildProcessCapability {
	return []ChildProcessCapability{
		{Name: "spawn", RuntimeSymbol: "jayess_child_process_spawn", Kind: "function"},
		{Name: "exec", RuntimeSymbol: "jayess_child_process_exec", Kind: "function"},
		{Name: "pipe", RuntimeSymbol: "jayess_child_process_pipe", Kind: "function"},
		{Name: "exitStatus", RuntimeSymbol: "jayess_child_process_exit_status", Kind: "function"},
		{Name: "signal", RuntimeSymbol: "jayess_child_process_signal", Kind: "function"},
		{Name: "cleanup", RuntimeSymbol: "jayess_child_process_cleanup", Kind: "function"},
	}
}

func HasChildProcessCapability(name string) bool {
	for _, capability := range ChildProcessCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
