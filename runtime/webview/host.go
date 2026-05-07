package webview

import "fmt"

type Host struct {
	nextID  int
	windows map[string]Window
	mounts  map[string]Mount
	events  []Event
	dialogs map[string]DialogRequest
	support Support
}

func NewHost() *Host {
	return &Host{
		windows: make(map[string]Window),
		mounts:  make(map[string]Mount),
		dialogs: make(map[string]DialogRequest),
		support: DefaultSupport(),
	}
}

func (host *Host) Support() Support {
	return host.support
}

func (host *Host) CreateWindow(title string, size Size) Window {
	host.nextID++
	window := Window{
		ID:    fmt.Sprintf("window-%d", host.nextID),
		Title: title,
		Size:  normalizeSize(size),
		State: WindowHidden,
	}
	host.windows[window.ID] = window
	host.events = append(host.events, Event{Kind: "window-opened", WindowID: window.ID})
	return window
}

func normalizeSize(size Size) Size {
	if size.Width <= 0 {
		size.Width = 800
	}
	if size.Height <= 0 {
		size.Height = 600
	}
	return size
}
