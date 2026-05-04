package runtime

type UtilCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func UtilCapabilities() []UtilCapability {
	return []UtilCapability{
		{Name: "format", RuntimeSymbol: "jayess_util_format", Kind: "function"},
		{Name: "inspect", RuntimeSymbol: "jayess_util_inspect", Kind: "function"},
	}
}

func HasUtilCapability(name string) bool {
	for _, capability := range UtilCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
