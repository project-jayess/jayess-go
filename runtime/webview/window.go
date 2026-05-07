package webview

import "fmt"

func (host *Host) Window(id string) (Window, bool) {
	window, ok := host.windows[id]
	return window, ok
}

func (host *Host) ShowWindow(id string) error {
	return host.updateWindowState(id, WindowShown)
}

func (host *Host) HideWindow(id string) error {
	return host.updateWindowState(id, WindowHidden)
}

func (host *Host) CloseWindow(id string) error {
	return host.updateWindowState(id, WindowClosed)
}

func (host *Host) SetWindowTitle(id string, title string) error {
	window, ok := host.windows[id]
	if !ok {
		return fmt.Errorf("unknown webview window %q", id)
	}
	window.Title = title
	host.windows[id] = window
	return nil
}

func (host *Host) SetWindowSize(id string, size Size) error {
	window, ok := host.windows[id]
	if !ok {
		return fmt.Errorf("unknown webview window %q", id)
	}
	window.Size = normalizeSize(size)
	host.windows[id] = window
	return nil
}

func (host *Host) updateWindowState(id string, state WindowState) error {
	window, ok := host.windows[id]
	if !ok {
		return fmt.Errorf("unknown webview window %q", id)
	}
	window.State = state
	host.windows[id] = window
	if state == WindowClosed {
		host.events = append(host.events, Event{Kind: "window-closed", WindowID: id})
	}
	return nil
}
