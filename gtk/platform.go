package gtk

type PlatformSupport struct {
	Platform     string
	PkgConfig    bool
	IncludeFlags []string
	LibraryFlags []string
	Supported    bool
	Diagnostic   string
}

func PlatformSupports() []PlatformSupport {
	return []PlatformSupport{
		{
			Platform:     "linux",
			PkgConfig:    true,
			IncludeFlags: []string{"gtk+-3.0"},
			LibraryFlags: []string{"gtk+-3.0"},
			Supported:    true,
		},
		{
			Platform:     "darwin",
			PkgConfig:    true,
			IncludeFlags: []string{"gtk+-3.0"},
			LibraryFlags: []string{"gtk+-3.0"},
			Supported:    true,
		},
		{
			Platform:     "windows",
			PkgConfig:    true,
			IncludeFlags: []string{"gtk+-3.0"},
			LibraryFlags: []string{"gtk+-3.0"},
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
		Diagnostic: "GTK platform support is not configured",
	}, false
}
