package runtime

type OSCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func OSCapabilities() []OSCapability {
	return []OSCapability{
		{Name: "platform", RuntimeSymbol: "jayess_os_platform", Kind: "function"},
		{Name: "arch", RuntimeSymbol: "jayess_os_arch", Kind: "function"},
		{Name: "tmpdir", RuntimeSymbol: "jayess_os_tmpdir", Kind: "function"},
		{Name: "hostname", RuntimeSymbol: "jayess_os_hostname", Kind: "function"},
		{Name: "uptime", RuntimeSymbol: "jayess_os_uptime", Kind: "function"},
		{Name: "cpus", RuntimeSymbol: "jayess_os_cpus", Kind: "function"},
		{Name: "memory", RuntimeSymbol: "jayess_os_memory", Kind: "function"},
		{Name: "userInfo", RuntimeSymbol: "jayess_os_user_info", Kind: "function"},
		{Name: "env", RuntimeSymbol: "jayess_os_env", Kind: "function"},
	}
}

func HasOSCapability(name string) bool {
	for _, capability := range OSCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
