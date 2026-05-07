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
		{Name: "listen", RuntimeSymbol: "jayess_http_server_listen", Kind: "function"},
		{Name: "close", RuntimeSymbol: "jayess_http_server_close", Kind: "function"},
		{Name: "address", RuntimeSymbol: "jayess_http_server_address", Kind: "function"},
		{Name: "on", RuntimeSymbol: "jayess_http_server_on", Kind: "function"},
		{Name: "addListener", RuntimeSymbol: "jayess_http_server_add_listener", Kind: "function"},
		{Name: "once", RuntimeSymbol: "jayess_http_server_once", Kind: "function"},
		{Name: "off", RuntimeSymbol: "jayess_http_server_off", Kind: "function"},
		{Name: "removeListener", RuntimeSymbol: "jayess_http_server_remove_listener", Kind: "function"},
		{Name: "removeAllListeners", RuntimeSymbol: "jayess_http_server_remove_all_listeners", Kind: "function"},
		{Name: "emit", RuntimeSymbol: "jayess_http_server_emit", Kind: "function"},
		{Name: "eventNames", RuntimeSymbol: "jayess_http_server_event_names", Kind: "function"},
		{Name: "listenerCount", RuntimeSymbol: "jayess_http_server_listener_count", Kind: "function"},
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
