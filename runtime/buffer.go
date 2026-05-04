package runtime

type BufferCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func BufferCapabilities() []BufferCapability {
	return []BufferCapability{
		{Name: "create", RuntimeSymbol: "jayess_buffer_create", Kind: "function"},
		{Name: "fromString", RuntimeSymbol: "jayess_buffer_from_string", Kind: "function"},
		{Name: "toString", RuntimeSymbol: "jayess_buffer_to_string", Kind: "function"},
		{Name: "slice", RuntimeSymbol: "jayess_buffer_slice", Kind: "function"},
		{Name: "copy", RuntimeSymbol: "jayess_buffer_copy", Kind: "function"},
		{Name: "readUInt16LE", RuntimeSymbol: "jayess_buffer_read_uint16_le", Kind: "function"},
		{Name: "writeUInt16LE", RuntimeSymbol: "jayess_buffer_write_uint16_le", Kind: "function"},
		{Name: "typedArrayView", RuntimeSymbol: "jayess_buffer_typed_array_view", Kind: "function"},
		{Name: "createReadStream", RuntimeSymbol: "jayess_buffer_create_read_stream", Kind: "function"},
		{Name: "createWriteStream", RuntimeSymbol: "jayess_buffer_create_write_stream", Kind: "function"},
	}
}

func HasBufferCapability(name string) bool {
	for _, capability := range BufferCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
