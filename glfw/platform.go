package glfw

type PlatformSupport struct {
	Platform     string
	IncludeDirs  []string
	LibraryFlags []string
	Frameworks   []string
	Supported    bool
	Diagnostic   string
}

func PlatformSupports() []PlatformSupport {
	return []PlatformSupport{
		{
			Platform:     "linux",
			LibraryFlags: []string{"-lglfw", "-lGL", "-ldl", "-lpthread"},
			Supported:    true,
		},
		{
			Platform:     "darwin",
			LibraryFlags: []string{"-lglfw"},
			Frameworks:   []string{"Cocoa", "IOKit", "CoreVideo", "OpenGL"},
			Supported:    true,
		},
		{
			Platform:     "windows",
			LibraryFlags: []string{"-lglfw3", "-lopengl32", "-lgdi32"},
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
		Diagnostic: "GLFW platform support is not configured",
	}, false
}
