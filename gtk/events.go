package gtk

type EventFeature string

const (
	ConnectSignal EventFeature = "connect-signal"
	ButtonClick   EventFeature = "button-click"
	InputChange   EventFeature = "input-change"
	WindowClose   EventFeature = "window-close"
	SafeCallback  EventFeature = "safe-callback"
)

func EventFeatures() []EventFeature {
	return []EventFeature{
		ConnectSignal,
		ButtonClick,
		InputChange,
		WindowClose,
		SafeCallback,
	}
}
