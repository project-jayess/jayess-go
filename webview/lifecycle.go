package webview

type LifecycleFeature string

const (
	CreateWindow   LifecycleFeature = "create-webview-window"
	DestroyWindow  LifecycleFeature = "destroy-webview-window"
	SetWindowTitle LifecycleFeature = "set-window-title"
	SetWindowSize  LifecycleFeature = "set-window-size"
	ShowWindow     LifecycleFeature = "show-window"
	HideWindow     LifecycleFeature = "hide-window"
	EnterEventLoop LifecycleFeature = "enter-event-loop"
	CleanShutdown  LifecycleFeature = "clean-shutdown"
)

func LifecycleFeatures() []LifecycleFeature {
	return []LifecycleFeature{
		CreateWindow,
		DestroyWindow,
		SetWindowTitle,
		SetWindowSize,
		ShowWindow,
		HideWindow,
		EnterEventLoop,
		CleanShutdown,
	}
}
