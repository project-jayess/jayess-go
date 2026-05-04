package runtime

type HTTPSCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func HTTPSCapabilities() []HTTPSCapability {
	return []HTTPSCapability{
		{Name: "createServer", RuntimeSymbol: "jayess_https_create_server", Kind: "function"},
		{Name: "request", RuntimeSymbol: "jayess_https_request", Kind: "function"},
		{Name: "loadCertificate", RuntimeSymbol: "jayess_https_load_certificate", Kind: "function"},
		{Name: "loadPrivateKey", RuntimeSymbol: "jayess_https_load_private_key", Kind: "function"},
		{Name: "trustStore", RuntimeSymbol: "jayess_https_trust_store", Kind: "function"},
		{Name: "verifyCertificate", RuntimeSymbol: "jayess_https_verify_certificate", Kind: "function"},
		{Name: "secureDefaults", RuntimeSymbol: "jayess_https_secure_defaults", Kind: "function"},
	}
}

func HasHTTPSCapability(name string) bool {
	for _, capability := range HTTPSCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
