package runtime

type DNSCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func DNSCapabilities() []DNSCapability {
	return []DNSCapability{
		{Name: "lookup", RuntimeSymbol: "jayess_dns_lookup", Kind: "function"},
		{Name: "reverse", RuntimeSymbol: "jayess_dns_reverse", Kind: "function"},
		{Name: "resolver", RuntimeSymbol: "jayess_dns_resolver", Kind: "function"},
		{Name: "isIP", RuntimeSymbol: "jayess_dns_is_ip", Kind: "function"},
		{Name: "parseIP", RuntimeSymbol: "jayess_dns_parse_ip", Kind: "function"},
	}
}

func HasDNSCapability(name string) bool {
	for _, capability := range DNSCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
