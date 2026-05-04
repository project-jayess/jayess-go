package webview

type PlatformSupport struct {
	Platform     string
	Backend      string
	LibraryFlags []string
	Frameworks   []string
	Supported    bool
	Diagnostic   string
}

func PlatformSupports() []PlatformSupport {
	return []PlatformSupport{
		{
			Platform:     "linux",
			Backend:      "webkit2gtk",
			LibraryFlags: []string{"-lgtk-3", "-lwebkit2gtk-4.1"},
			Supported:    true,
		},
		{
			Platform:   "darwin",
			Backend:    "wkwebview",
			Frameworks: []string{"Cocoa", "WebKit"},
			Supported:  true,
		},
		{
			Platform:     "windows",
			Backend:      "webview2",
			LibraryFlags: []string{"-lole32", "-lcomctl32"},
			Supported:    true,
		},
	}
}

func PlatformSupportFor(platform string) (PlatformSupport, bool) {
	for _, support := range PlatformSupports() {
		if support.Platform == platform {
			return support, true
		}
	}
	return PlatformSupport{
		Platform:   platform,
		Diagnostic: "webview platform support is not configured",
	}, false
}
