package webview

import "fmt"

const (
	OpenDialogKind = "open"
	SaveDialogKind = "save"
)

func (host *Host) OpenFileDialog(windowID string, request DialogRequest) error {
	return host.queueDialog(windowID, OpenDialogKind, request)
}

func (host *Host) SaveFileDialog(windowID string, request DialogRequest) error {
	return host.queueDialog(windowID, SaveDialogKind, request)
}

func (host *Host) CompleteDialog(windowID string, result DialogResult) error {
	if _, ok := host.dialogs[windowID]; !ok {
		return fmt.Errorf("no pending webview dialog for %q", windowID)
	}
	delete(host.dialogs, windowID)
	host.events = append(host.events, Event{
		Kind:     "dialog-result",
		WindowID: windowID,
		Result:   result,
	})
	return nil
}

func (host *Host) PendingDialog(windowID string) (DialogRequest, bool) {
	request, ok := host.dialogs[windowID]
	return request, ok
}

func (host *Host) queueDialog(windowID string, kind string, request DialogRequest) error {
	if _, ok := host.windows[windowID]; !ok {
		return fmt.Errorf("unknown webview window %q", windowID)
	}
	request.Kind = kind
	host.dialogs[windowID] = request
	return nil
}
