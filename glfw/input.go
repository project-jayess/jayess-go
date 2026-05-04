package glfw

type InputFeature string

const (
	KeyboardCallback       InputFeature = "keyboard-callback"
	MouseButtonCallback    InputFeature = "mouse-button-callback"
	CursorPositionCallback InputFeature = "cursor-position-callback"
	ScrollCallback         InputFeature = "scroll-callback"
	GamepadJoystickInput   InputFeature = "gamepad-joystick-input"
)

func InputFeatures() []InputFeature {
	return []InputFeature{
		KeyboardCallback,
		MouseButtonCallback,
		CursorPositionCallback,
		ScrollCallback,
		GamepadJoystickInput,
	}
}
