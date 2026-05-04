package libuv

type LoopFeature string

const (
	CreateLoop       LoopFeature = "create-loop"
	RunLoop          LoopFeature = "run-loop"
	StopLoop         LoopFeature = "stop-loop"
	CloseLoop        LoopFeature = "close-loop"
	TimerCoexistence LoopFeature = "timer-coexistence"
	MicrotaskPolicy  LoopFeature = "microtask-policy"
)

func LoopFeatures() []LoopFeature {
	return []LoopFeature{
		CreateLoop,
		RunLoop,
		StopLoop,
		CloseLoop,
		TimerCoexistence,
		MicrotaskPolicy,
	}
}
