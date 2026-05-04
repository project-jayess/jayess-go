package webview

type BridgeFeature string

const (
	ExposeJayessFunction   BridgeFeature = "expose-jayess-function"
	ReceiveJavaScriptEvent BridgeFeature = "receive-javascript-event"
	SafeStringJSONBoundary BridgeFeature = "safe-string-json-boundary"
	SafeCallbackLifetime   BridgeFeature = "safe-callback-lifetime"
	BridgeErrorPropagation BridgeFeature = "bridge-error-propagation"
)

func BridgeFeatures() []BridgeFeature {
	return []BridgeFeature{
		ExposeJayessFunction,
		ReceiveJavaScriptEvent,
		SafeStringJSONBoundary,
		SafeCallbackLifetime,
		BridgeErrorPropagation,
	}
}
