package runtime

type StreamCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func StreamCapabilities() []StreamCapability {
	return []StreamCapability{
		{Name: "readable", RuntimeSymbol: "jayess_stream_readable", Kind: "function"},
		{Name: "writable", RuntimeSymbol: "jayess_stream_writable", Kind: "function"},
		{Name: "duplex", RuntimeSymbol: "jayess_stream_duplex", Kind: "function"},
		{Name: "transform", RuntimeSymbol: "jayess_stream_transform", Kind: "function"},
		{Name: "pipe", RuntimeSymbol: "jayess_stream_pipe", Kind: "function"},
		{Name: "awaitDrain", RuntimeSymbol: "jayess_stream_await_drain", Kind: "function"},
	}
}

func HasStreamCapability(name string) bool {
	for _, capability := range StreamCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
