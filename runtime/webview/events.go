package webview

func (host *Host) EmitHostMessage(windowID string, message string) {
	host.events = append(host.events, Event{
		Kind:     "host-message",
		WindowID: windowID,
		Message:  message,
	})
}

func (host *Host) QueueFileDrop(windowID string, paths []string) {
	host.events = append(host.events, Event{
		Kind:     "file-drop",
		WindowID: windowID,
		Paths:    append([]string{}, paths...),
	})
}

func (host *Host) DrainEvents() []Event {
	events := append([]Event{}, host.events...)
	host.events = nil
	return events
}
