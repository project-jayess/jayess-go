package webview

import "fmt"

func (host *Host) MountContent(windowID string, mount Mount) error {
	if _, ok := host.windows[windowID]; !ok {
		return fmt.Errorf("unknown webview window %q", windowID)
	}
	host.mounts[windowID] = mount
	host.events = append(host.events, Event{Kind: "navigation", WindowID: windowID, Message: mount.Kind})
	return nil
}

func (host *Host) MountedContent(windowID string) (Mount, bool) {
	mount, ok := host.mounts[windowID]
	return mount, ok
}
