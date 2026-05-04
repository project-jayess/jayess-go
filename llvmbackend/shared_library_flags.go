package llvmbackend

func SharedLibraryLinkFlags(target TargetConfig) []string {
	switch targetPlatform(target) {
	case "darwin":
		return []string{"-dynamiclib"}
	case "windows":
		return []string{"-shared"}
	default:
		return []string{"-shared"}
	}
}

func targetPlatform(target TargetConfig) string {
	switch target.Name {
	case "macos-x64", "macos-arm64":
		return "darwin"
	case "windows-x64":
		return "windows"
	default:
		return "linux"
	}
}
