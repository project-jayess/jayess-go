package runtime

type WorkerCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func WorkerCapabilities() []WorkerCapability {
	return []WorkerCapability{
		{Name: "thread", RuntimeSymbol: "jayess_worker_thread", Kind: "function"},
		{Name: "postMessage", RuntimeSymbol: "jayess_worker_post_message", Kind: "function"},
		{Name: "onMessage", RuntimeSymbol: "jayess_worker_on_message", Kind: "function"},
		{Name: "sharedMemory", RuntimeSymbol: "jayess_worker_shared_memory", Kind: "function"},
		{Name: "atomicLoad", RuntimeSymbol: "jayess_worker_atomic_load", Kind: "function"},
		{Name: "atomicStore", RuntimeSymbol: "jayess_worker_atomic_store", Kind: "function"},
	}
}

func HasWorkerCapability(name string) bool {
	for _, capability := range WorkerCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
