package webview

type EventKind string

const (
	WindowOpenedEvent EventKind = "window-opened"
	WindowClosedEvent EventKind = "window-closed"
	NavigationEvent   EventKind = "navigation"
	HostMessageEvent  EventKind = "host-message"
	DialogResultEvent EventKind = "dialog-result"
	FileDropEvent     EventKind = "file-drop"
)

func EventKinds() []EventKind {
	return []EventKind{
		WindowOpenedEvent,
		WindowClosedEvent,
		NavigationEvent,
		HostMessageEvent,
		DialogResultEvent,
		FileDropEvent,
	}
}
