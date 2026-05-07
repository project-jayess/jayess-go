package webview

import "path/filepath"

type BackendRequirement struct {
	TargetName            string
	Backend               string
	RedistributableAssets []string
	SystemPrerequisites   []string
}

func BackendRequirements() []BackendRequirement {
	return []BackendRequirement{
		{
			TargetName:            "windows-x64",
			Backend:               "webview2-fixed-runtime",
			RedistributableAssets: []string{filepath.Join("webview", "windows", "WebView2Loader.dll")},
		},
		{
			TargetName:          "darwin-arm64",
			Backend:             "wkwebview",
			SystemPrerequisites: []string{"Cocoa.framework", "WebKit.framework"},
		},
		{
			TargetName:          "darwin-x64",
			Backend:             "wkwebview",
			SystemPrerequisites: []string{"Cocoa.framework", "WebKit.framework"},
		},
		{
			TargetName:          "linux-x64",
			Backend:             "webkitgtk",
			SystemPrerequisites: []string{"gtk", "webkit2gtk"},
		},
	}
}

func BackendRequirementForTarget(targetName string) (BackendRequirement, bool) {
	for _, requirement := range BackendRequirements() {
		if requirement.TargetName == targetName {
			return requirement, true
		}
	}
	return BackendRequirement{}, false
}
