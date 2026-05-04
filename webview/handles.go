package webview

type HandleKind string

const (
	WebviewHandle HandleKind = "webview"
	WindowHandle  HandleKind = "native-window"
	BridgeHandle  HandleKind = "bridge-callback"
	ServerHandle  HandleKind = "embedded-http-server"
)

type HandleRule struct {
	Kind     HandleKind
	Managed  bool
	Closable bool
	Nullable bool
}

func HandleRules() []HandleRule {
	return []HandleRule{
		{Kind: WebviewHandle, Managed: true, Closable: true},
		{Kind: WindowHandle, Managed: true, Closable: true, Nullable: true},
		{Kind: BridgeHandle, Managed: true, Closable: true},
		{Kind: ServerHandle, Managed: true, Closable: true, Nullable: true},
	}
}

func SupportsHandle(kind HandleKind) bool {
	for _, rule := range HandleRules() {
		if rule.Kind == kind && rule.Managed {
			return true
		}
	}
	return false
}
