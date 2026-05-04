package runtime

type CompressionCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func CompressionCapabilities() []CompressionCapability {
	return []CompressionCapability{
		{Name: "gzip", RuntimeSymbol: "jayess_compression_gzip", Kind: "function"},
		{Name: "gunzip", RuntimeSymbol: "jayess_compression_gunzip", Kind: "function"},
		{Name: "deflate", RuntimeSymbol: "jayess_compression_deflate", Kind: "function"},
		{Name: "inflate", RuntimeSymbol: "jayess_compression_inflate", Kind: "function"},
		{Name: "brotliCompress", RuntimeSymbol: "jayess_compression_brotli_compress", Kind: "function"},
		{Name: "brotliDecompress", RuntimeSymbol: "jayess_compression_brotli_decompress", Kind: "function"},
		{Name: "createCompressStream", RuntimeSymbol: "jayess_compression_create_compress_stream", Kind: "function"},
		{Name: "createDecompressStream", RuntimeSymbol: "jayess_compression_create_decompress_stream", Kind: "function"},
	}
}

func HasCompressionCapability(name string) bool {
	for _, capability := range CompressionCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
