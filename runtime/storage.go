package runtime

type StorageCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func StorageCapabilities() []StorageCapability {
	return []StorageCapability{
		{Name: "open", RuntimeSymbol: "jayess_storage_open", Kind: "function"},
		{Name: "close", RuntimeSymbol: "jayess_storage_close", Kind: "function"},
		{Name: "get", RuntimeSymbol: "jayess_storage_get", Kind: "function"},
		{Name: "put", RuntimeSymbol: "jayess_storage_put", Kind: "function"},
		{Name: "delete", RuntimeSymbol: "jayess_storage_delete", Kind: "function"},
		{Name: "scan", RuntimeSymbol: "jayess_storage_scan", Kind: "function"},
	}
}

func HasStorageCapability(name string) bool {
	for _, capability := range StorageCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
