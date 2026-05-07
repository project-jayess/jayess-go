package webview

func UnsupportedPlatformDiagnostic(platform string) string {
	return "webview platform support is not configured for " + platform
}

func UnsupportedCapabilityDiagnostic(capability string) string {
	return "webview host capability is not supported: " + capability
}
