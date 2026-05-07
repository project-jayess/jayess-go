package runtime

import "net"

type TCPCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func TCPCapabilities() []TCPCapability {
	return []TCPCapability{
		{Name: "client", RuntimeSymbol: "jayess_tcp_client", Kind: "function"},
		{Name: "server", RuntimeSymbol: "jayess_tcp_server", Kind: "function"},
		{Name: "connect", RuntimeSymbol: "jayess_tcp_connect", Kind: "function"},
		{Name: "listen", RuntimeSymbol: "jayess_tcp_listen", Kind: "function"},
		{Name: "accept", RuntimeSymbol: "jayess_tcp_accept", Kind: "function"},
		{Name: "read", RuntimeSymbol: "jayess_tcp_read", Kind: "function"},
		{Name: "write", RuntimeSymbol: "jayess_tcp_write", Kind: "function"},
		{Name: "close", RuntimeSymbol: "jayess_tcp_close", Kind: "function"},
		{Name: "lastError", RuntimeSymbol: "jayess_tcp_last_error", Kind: "function"},
		{Name: "withTimeout", RuntimeSymbol: "jayess_tcp_with_timeout", Kind: "function"},
		{Name: "awaitDrain", RuntimeSymbol: "jayess_tcp_await_drain", Kind: "function"},
	}
}

func HasTCPCapability(name string) bool {
	for _, capability := range TCPCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}

func listenTCP(address string) (net.Listener, error) {
	return net.Listen("tcp", address)
}
