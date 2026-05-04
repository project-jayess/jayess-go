package runtime

type TLSCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func TLSCapabilities() []TLSCapability {
	return []TLSCapability{
		{Name: "client", RuntimeSymbol: "jayess_tls_client", Kind: "function"},
		{Name: "server", RuntimeSymbol: "jayess_tls_server", Kind: "function"},
		{Name: "certificate", RuntimeSymbol: "jayess_tls_certificate", Kind: "function"},
		{Name: "withALPN", RuntimeSymbol: "jayess_tls_with_alpn", Kind: "function"},
		{Name: "verifyHostname", RuntimeSymbol: "jayess_tls_verify_hostname", Kind: "function"},
	}
}

func HasTLSCapability(name string) bool {
	for _, capability := range TLSCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
