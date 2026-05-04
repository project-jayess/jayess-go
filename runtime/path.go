package runtime

type PathCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func PathCapabilities() []PathCapability {
	return []PathCapability{
		{Name: "join", RuntimeSymbol: "jayess_path_join", Kind: "function"},
		{Name: "resolve", RuntimeSymbol: "jayess_path_resolve", Kind: "function"},
		{Name: "normalize", RuntimeSymbol: "jayess_path_normalize", Kind: "function"},
		{Name: "basename", RuntimeSymbol: "jayess_path_basename", Kind: "function"},
		{Name: "dirname", RuntimeSymbol: "jayess_path_dirname", Kind: "function"},
		{Name: "extname", RuntimeSymbol: "jayess_path_extname", Kind: "function"},
		{Name: "relative", RuntimeSymbol: "jayess_path_relative", Kind: "function"},
	}
}

func HasPathCapability(name string) bool {
	for _, capability := range PathCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
