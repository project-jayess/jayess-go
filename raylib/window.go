package raylib

type WindowFeature string

const (
	InitializeRaylib       WindowFeature = "initialize-raylib"
	CreateWindow           WindowFeature = "create-window"
	SetWindowTitle         WindowFeature = "set-window-title"
	SetWindowSize          WindowFeature = "set-window-size"
	WindowShouldClose      WindowFeature = "window-should-close"
	CloseWindow            WindowFeature = "close-window"
	FrameUpdateLoop        WindowFeature = "frame-update-loop"
	FullscreenModeSwitch   WindowFeature = "fullscreen-mode-switch"
	RuntimeLoopCoexistence WindowFeature = "runtime-loop-coexistence"
)

func WindowFeatures() []WindowFeature {
	return []WindowFeature{
		InitializeRaylib,
		CreateWindow,
		SetWindowTitle,
		SetWindowSize,
		WindowShouldClose,
		CloseWindow,
		FrameUpdateLoop,
		FullscreenModeSwitch,
		RuntimeLoopCoexistence,
	}
}
