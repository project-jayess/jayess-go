package webview

type Support struct {
	PackageImport          string
	RuntimeAssetFile       string
	RuntimeAssetOutput     string
	UsesInternalRuntime    bool
	RequiresPackageInstall bool
	RequiresEndUserInstall bool
	ShipsThirdPartyGUILibs bool
}

func DefaultSupport() Support {
	return Support{
		PackageImport:          PackageImport,
		RuntimeAssetFile:       RuntimeAssetFile,
		RuntimeAssetOutput:     RuntimeAssetOutputPath(),
		UsesInternalRuntime:    true,
		RequiresPackageInstall: false,
		RequiresEndUserInstall: false,
		ShipsThirdPartyGUILibs: false,
	}
}
