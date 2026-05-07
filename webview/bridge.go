package webview

type BridgeFeature string

const (
	ExposeJayessFunction   BridgeFeature = "expose-jayess-function"
	ReceiveJavaScriptEvent BridgeFeature = "receive-javascript-event"
	SafeStringJSONBoundary BridgeFeature = "safe-string-json-boundary"
	SafeCallbackLifetime   BridgeFeature = "safe-callback-lifetime"
	BridgeErrorPropagation BridgeFeature = "bridge-error-propagation"
)

type HostCallKind string

const (
	CreateWindowCall    HostCallKind = "create-window"
	MountContentCall    HostCallKind = "mount-content"
	DispatchEventCall   HostCallKind = "dispatch-event"
	EmitHostMessageCall HostCallKind = "emit-host-message"
)

type BridgeContract struct {
	Calls  []HostCallKind
	Events []EventKind
}

func BridgeFeatures() []BridgeFeature {
	return []BridgeFeature{
		ExposeJayessFunction,
		ReceiveJavaScriptEvent,
		SafeStringJSONBoundary,
		SafeCallbackLifetime,
		BridgeErrorPropagation,
	}
}

func DefaultBridgeContract() BridgeContract {
	return BridgeContract{
		Calls: []HostCallKind{
			CreateWindowCall,
			MountContentCall,
			DispatchEventCall,
			EmitHostMessageCall,
		},
		Events: []EventKind{
			WindowOpenedEvent,
			WindowClosedEvent,
			HostMessageEvent,
			DialogResultEvent,
			FileDropEvent,
		},
	}
}
