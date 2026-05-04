package webview

type ContentFeature string

const (
	LoadInlineHTML    ContentFeature = "load-inline-html"
	LoadLocalFile     ContentFeature = "load-local-file"
	NavigateToURL     ContentFeature = "navigate-to-url"
	ServeLocalHTTPApp ContentFeature = "serve-local-http-app"
	InjectJavaScript  ContentFeature = "inject-javascript"
)

func ContentFeatures() []ContentFeature {
	return []ContentFeature{
		LoadInlineHTML,
		LoadLocalFile,
		NavigateToURL,
		ServeLocalHTTPApp,
		InjectJavaScript,
	}
}
