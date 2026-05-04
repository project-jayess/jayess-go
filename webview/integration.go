package webview

type IntegrationFeature string

const (
	NativeHTTPServerIntegration IntegrationFeature = "native-http-server"
	WorkerThreadIntegration     IntegrationFeature = "worker-thread"
	FilesystemPathIntegration   IntegrationFeature = "filesystem-path"
	GLFWHostIntegration         IntegrationFeature = "glfw-host"
	GTKHostIntegration          IntegrationFeature = "gtk-host"
)

func IntegrationFeatures() []IntegrationFeature {
	return []IntegrationFeature{
		NativeHTTPServerIntegration,
		WorkerThreadIntegration,
		FilesystemPathIntegration,
		GLFWHostIntegration,
		GTKHostIntegration,
	}
}
