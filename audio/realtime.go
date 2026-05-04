package audio

type RealtimeFeature string

const (
	LowLatencyPlayback RealtimeFeature = "low-latency-playback"
	UnderrunHandling   RealtimeFeature = "underrun-handling"
	DeviceLossHandling RealtimeFeature = "device-loss-handling"
	ThreadSafeCallback RealtimeFeature = "thread-safe-callback"
	WorkerSync         RealtimeFeature = "worker-sync"
)

func RealtimeFeatures() []RealtimeFeature {
	return []RealtimeFeature{
		LowLatencyPlayback,
		UnderrunHandling,
		DeviceLossHandling,
		ThreadSafeCallback,
		WorkerSync,
	}
}
