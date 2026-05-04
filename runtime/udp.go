package runtime

type UDPCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func UDPCapabilities() []UDPCapability {
	return []UDPCapability{
		{Name: "socket", RuntimeSymbol: "jayess_udp_socket", Kind: "function"},
		{Name: "send", RuntimeSymbol: "jayess_udp_send", Kind: "function"},
		{Name: "receive", RuntimeSymbol: "jayess_udp_receive", Kind: "function"},
		{Name: "bind", RuntimeSymbol: "jayess_udp_bind", Kind: "function"},
		{Name: "joinMulticast", RuntimeSymbol: "jayess_udp_join_multicast", Kind: "function"},
		{Name: "setBroadcast", RuntimeSymbol: "jayess_udp_set_broadcast", Kind: "function"},
	}
}

func HasUDPCapability(name string) bool {
	for _, capability := range UDPCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
