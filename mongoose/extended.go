package mongoose

type ExtendedFeature string

const (
	StaticFileServing ExtendedFeature = "static-file-serving"
	RouteDispatch     ExtendedFeature = "route-dispatch"
	ChunkedStreaming  ExtendedFeature = "chunked-streaming"
	HTTPSServing      ExtendedFeature = "https-serving"
	WebSocketUpgrade  ExtendedFeature = "websocket-upgrade"
	WebviewAppContent ExtendedFeature = "webview-app-content"
)

func ExtendedFeatures() []ExtendedFeature {
	return []ExtendedFeature{
		StaticFileServing,
		RouteDispatch,
		ChunkedStreaming,
		HTTPSServing,
		WebSocketUpgrade,
		WebviewAppContent,
	}
}
