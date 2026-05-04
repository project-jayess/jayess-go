package runtime

type HTTPCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func HTTPCapabilities() []HTTPCapability {
	return []HTTPCapability{
		{Name: "createServer", RuntimeSymbol: "jayess_http_create_server", Kind: "function"},
		{Name: "request", RuntimeSymbol: "jayess_http_request", Kind: "function"},
		{Name: "requestObject", RuntimeSymbol: "jayess_http_request_object", Kind: "function"},
		{Name: "responseObject", RuntimeSymbol: "jayess_http_response_object", Kind: "function"},
		{Name: "headers", RuntimeSymbol: "jayess_http_headers", Kind: "function"},
		{Name: "status", RuntimeSymbol: "jayess_http_status", Kind: "function"},
		{Name: "readBody", RuntimeSymbol: "jayess_http_read_body", Kind: "function"},
		{Name: "writeBody", RuntimeSymbol: "jayess_http_write_body", Kind: "function"},
		{Name: "streamBody", RuntimeSymbol: "jayess_http_stream_body", Kind: "function"},
		{Name: "keepAlive", RuntimeSymbol: "jayess_http_keep_alive", Kind: "function"},
		{Name: "withTimeout", RuntimeSymbol: "jayess_http_with_timeout", Kind: "function"},
	}
}

func HasHTTPCapability(name string) bool {
	for _, capability := range HTTPCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
