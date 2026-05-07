package webview

type WindowState string

const (
	WindowHidden WindowState = "hidden"
	WindowShown  WindowState = "shown"
	WindowClosed WindowState = "closed"
)

type Size struct {
	Width  int
	Height int
}

type Window struct {
	ID    string
	Title string
	Size  Size
	State WindowState
}

type Asset struct {
	Path string
	Kind string
}

type Mount struct {
	Kind    string
	HTML    string
	CSS     string
	Script  string
	Assets  []Asset
	Entry   string
	Mutable bool
}

type DialogRequest struct {
	Kind        string
	Title       string
	DefaultPath string
	Filters     []string
}

type DialogResult struct {
	Accepted bool
	Path     string
}

type Event struct {
	Kind     string
	WindowID string
	Message  string
	Paths    []string
	Result   DialogResult
}
