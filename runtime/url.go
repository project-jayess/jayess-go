package runtime

type URLCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func URLCapabilities() []URLCapability {
	return []URLCapability{
		{Name: "parse", RuntimeSymbol: "jayess_url_parse", Kind: "function"},
		{Name: "format", RuntimeSymbol: "jayess_url_format", Kind: "function"},
		{Name: "parseQuery", RuntimeSymbol: "jayess_url_parse_query", Kind: "function"},
		{Name: "stringifyQuery", RuntimeSymbol: "jayess_url_stringify_query", Kind: "function"},
		{Name: "encode", RuntimeSymbol: "jayess_url_encode", Kind: "function"},
		{Name: "decode", RuntimeSymbol: "jayess_url_decode", Kind: "function"},
		{Name: "fileURLToPath", RuntimeSymbol: "jayess_url_file_url_to_path", Kind: "function"},
		{Name: "pathToFileURL", RuntimeSymbol: "jayess_url_path_to_file_url", Kind: "function"},
	}
}

func HasURLCapability(name string) bool {
	for _, capability := range URLCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
