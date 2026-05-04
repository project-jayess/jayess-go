package mongoose

type HandleKind string

const (
	ManagerHandle    HandleKind = "mg_mgr"
	ConnectionHandle HandleKind = "mg_connection"
	RequestHandle    HandleKind = "mg_http_message"
	WebSocketHandle  HandleKind = "mg_ws_message"
)

type HandleRule struct {
	Kind          HandleKind
	Managed       bool
	Closable      bool
	CallbackOwned bool
	Nullable      bool
}

func HandleRules() []HandleRule {
	return []HandleRule{
		{Kind: ManagerHandle, Managed: true, Closable: true},
		{Kind: ConnectionHandle, Managed: true, Closable: true, CallbackOwned: true, Nullable: true},
		{Kind: RequestHandle, Managed: false, CallbackOwned: true, Nullable: true},
		{Kind: WebSocketHandle, Managed: false, CallbackOwned: true, Nullable: true},
	}
}

func SupportsHandle(kind HandleKind) bool {
	for _, rule := range HandleRules() {
		if rule.Kind == kind {
			return true
		}
	}
	return false
}
