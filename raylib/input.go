package raylib

type InputFeature string

const (
	KeyboardInput      InputFeature = "keyboard-input"
	MouseInput         InputFeature = "mouse-input"
	GamepadInput       InputFeature = "gamepad-input"
	FrameDeltaTiming   InputFeature = "frame-delta-timing"
	TargetFPSTiming    InputFeature = "target-fps-timing"
	WindowModeSwitch   InputFeature = "window-mode-switch"
	RenderLoopControls InputFeature = "render-loop-controls"
)

func InputFeatures() []InputFeature {
	return []InputFeature{
		KeyboardInput,
		MouseInput,
		GamepadInput,
		FrameDeltaTiming,
		TargetFPSTiming,
		WindowModeSwitch,
		RenderLoopControls,
	}
}
