package gtk

type LifecycleFeature string

const (
	InitializeRuntime LifecycleFeature = "initialize-runtime"
	CreateApplication LifecycleFeature = "create-application"
	CreateWindow      LifecycleFeature = "create-window"
	EnterMainLoop     LifecycleFeature = "enter-main-loop"
	QuitMainLoop      LifecycleFeature = "quit-main-loop"
	CleanShutdown     LifecycleFeature = "clean-shutdown"
)

func LifecycleFeatures() []LifecycleFeature {
	return []LifecycleFeature{
		InitializeRuntime,
		CreateApplication,
		CreateWindow,
		EnterMainLoop,
		QuitMainLoop,
		CleanShutdown,
	}
}
