package libuv

type IOFeature string

const (
	TCPIntegration     IOFeature = "tcp-integration"
	UDPIntegration     IOFeature = "udp-integration"
	FilesystemAsyncOps IOFeature = "filesystem-async-ops"
	PollingWatcher     IOFeature = "polling-watcher"
	ProcessSpawn       IOFeature = "process-spawn"
	SignalWatcher      IOFeature = "signal-watcher"
)

func IOFeatures() []IOFeature {
	return []IOFeature{
		TCPIntegration,
		UDPIntegration,
		FilesystemAsyncOps,
		PollingWatcher,
		ProcessSpawn,
		SignalWatcher,
	}
}
